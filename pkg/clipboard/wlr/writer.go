package wlr

import (
	"errors"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/labi-le/belphegor/pkg/mime"
	"github.com/rs/zerolog"
)

const (
	writeTimeout = 5 * time.Second
)

type writer struct {
	*preset
	logger zerolog.Logger
	reader *reader

	mu           sync.Mutex
	activeSource *controlSource
	closed       bool
}

func newWriter(preset *preset, reader *reader, log zerolog.Logger) *writer {
	return &writer{
		preset: preset,
		reader: reader,
		logger: log.With().Str("component", "writer").Logger(),
	}
}

type sourceListener struct {
	data   []byte
	source *controlSource
	logger zerolog.Logger
	once   sync.Once
}

func (s *sourceListener) Send(_ string, f *os.File) {
	go func(f *os.File) {
		defer f.Close()

		ctxLog := s.logger.With().Str("op", "Send").Logger()

		timer := time.AfterFunc(writeTimeout, func() { f.Close() })
		defer timer.Stop()

		var total int
		var writeErr error

		for total < len(s.data) {
			n, err := f.Write(s.data[total:])
			if n > 0 {
				total += n
			}
			if err != nil {
				writeErr = err
				break
			}
		}

		if writeErr != nil && !isExpectedSocketError(writeErr) {
			ctxLog.Trace().Err(writeErr).Int("written", total).Msg("write failed")
			return
		}
	}(f)
}

func isExpectedSocketError(err error) bool {
	if errors.Is(err, syscall.EPIPE) {
		return true
	}
	if errors.Is(err, syscall.ECONNRESET) {
		return true
	}
	if errors.Is(err, syscall.EDESTADDRREQ) {
		return true
	}
	if errors.Is(err, syscall.EBADF) {
		return true
	}
	if errors.Is(err, os.ErrClosed) {
		return true
	}
	return false
}

func (s *sourceListener) Cancelled() {
	s.once.Do(func() {
		if s.source != nil {
			s.source.Destroy()
		}
	})
}

func (w *writer) Write(t mime.Type, p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		w.logger.Warn().Msg("writer is closed, ignoring write")
		return 0, errors.New("writer is closed")
	}

	if w.deviceManager == nil {
		w.logger.Error().Msg("data control manager not initialized")
		return 0, errors.New("data control manager not initialized")
	}

	source := w.deviceManager.CreateDataSource()

	dataCopy := make([]byte, len(p))
	copy(dataCopy, p)
	listener := &sourceListener{
		data:   dataCopy,
		source: source,
		logger: w.logger,
	}
	source.Listener = listener

	for _, o := range w.convertMimeType(t) {
		source.Offer(o)
	}

	if w.reader != nil {
		w.reader.Suppress()
	}

	w.device.SetSelection(source)
	w.activeSource = source

	return len(p), nil
}

func (w *writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.logger.Debug().Msg("Closing writer")
	w.closed = true

	//goland:noinspection GoMaybeNil
	w.device.SetSelection(nil)
	w.activeSource = nil

	return nil
}

var (
	offerText = []string{
		"text/plain;charset=utf-8",
		"text/plain",
		"TEXT",
		"STRING",
		"UTF8_STRING",
	}

	offerPath = append([]string{
		"text/uri-list",
		"x-special/gnome-copied-files",
	}, offerText...)

	offerImage = []string{
		"image/png",
		"image/jpeg",
		"image/gif",
		"image/bmp",
		"image/webp",
	}
	offerBinary = []string{"application/octet-stream"}
)

func (w *writer) convertMimeType(t mime.Type) []string {
	switch t {
	case mime.TypeText:
		return offerText
	case mime.TypePath:
		return offerPath
	case mime.TypeImage:
		return offerImage
	default:
		return offerBinary
	}
}
