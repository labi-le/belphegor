package wlr

import (
	"bytes"
	"errors"
	"os"
	"sync"
	"syscall"
	"unicode/utf8"

	"github.com/rs/zerolog"
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

func (s *sourceListener) Send(mime string, f *os.File) {
	ctxLog := s.logger.With().Str("op", "Send").Logger()

	total := 0
	fd := int(f.Fd())
	for total < len(s.data) {
		n, err := syscall.Write(fd, s.data[total:])
		if err != nil {
			if errors.Is(err, syscall.EPIPE) {
				ctxLog.Debug().
					Int("written", total).
					Int("total", len(s.data)).
					Msg("reader closed pipe early (normal)")
			} else {
				ctxLog.Error().Err(err).Int("written", total).Msg("failed to write clipboard data")
			}
			break
		}
		total += n
	}

	if err := f.Close(); err != nil {
		ctxLog.Error().Err(err).Msg("failed to close file")
	}
}

func (s *sourceListener) Cancelled() {}

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
		w.reader.IgnoreNextSelection()
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

	w.device.SetSelection(new(controlSource))
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
