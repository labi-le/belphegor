package wlr

import (
	"bytes"
	"errors"
	"os"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"

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
	if s.source != nil {
		s.source.Destroy()
	}
}

func (w *writer) Write(p []byte) (n int, err error) {
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

	typ := mimeType(p)
	if typ == "" || utf8.Valid(p) {
		source.Offer("text/plain;charset=utf-8")
		source.Offer("text/plain")
		source.Offer("TEXT")
		source.Offer("STRING")
		source.Offer("UTF8_STRING")
	} else {
		source.Offer(typ)
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

func mimeType(data []byte) string {
	switch {
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x89, 0x50, 0x4E, 0x47}):
		return "image/png"
	case len(data) >= 2 && bytes.Equal(data[:2], []byte{0xFF, 0xD8}):
		return "image/jpeg"
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x47, 0x49, 0x46, 0x38}):
		return "image/gif"
	case len(data) >= 2 && bytes.Equal(data[:2], []byte{0x42, 0x4D}):
		return "image/bmp"
	case len(data) >= 12 && bytes.Equal(data[8:12], []byte("WEBP")):
		return "image/webp"
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x25, 0x50, 0x44, 0x46}):
		return "application/pdf"
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x50, 0x4B, 0x03, 0x04}):
		return "application/zip"
	default:
		return ""
	}
}
