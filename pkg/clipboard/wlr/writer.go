package wlr

import (
	"bytes"
	"errors"
	"github.com/rs/zerolog"
	"sync"
	"syscall"
)

type writer struct {
	*preset
	logger zerolog.Logger

	mu           sync.Mutex
	activeSource *ZwlrDataControlSourceV1
}

func newWriter(preset *preset, log zerolog.Logger) *writer {
	return &writer{
		preset: preset,
		logger: log.With().Str("component", "writer").Logger(),
	}
}

type sourceListener struct {
	data   []byte
	source *ZwlrDataControlSourceV1
	logger zerolog.Logger
}

func (s *sourceListener) Send(mime string, fd int) {
	ctxLog := s.logger.With().Str("op", "Send").Logger()
	ctxLog.Trace().Msgf("writing %d bytes to fd: %d", len(s.data), fd)

	_, err := syscall.Write(fd, s.data)
	if err != nil {
		ctxLog.Error().AnErr("syscall.Write", err).Msg("failed to write clipboard data")
	}

	if err := syscall.Close(fd); err != nil {
		ctxLog.Error().AnErr("syscall.Close", err).Msg("failed to close fd")
	}
}

func (s *sourceListener) Cancelled() {
	s.logger.Trace().Str("op", "Cancelled").Msgf("source %d cancelled", s.source.ID())
	s.source.Destroy()
}

func (w *writer) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.deviceManager == nil {
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
	if typ == "" || mimeTypeText(p) {
		source.Offer("text/plain")
		source.Offer("TEXT")
		source.Offer("STRING")
		source.Offer("UTF8_STRING")
	} else {
		source.Offer(typ)
	}

	w.device.SetSelection(source)
	w.activeSource = source

	return len(p), nil
}

func (w *writer) Close() error {
	return w.client.Close()
}

func mimeType(data []byte) string {
	switch {
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x89, 0x50, 0x4E, 0x47}): // PNG
		return "image/png"
	case len(data) >= 2 && bytes.Equal(data[:2], []byte{0xFF, 0xD8}): // JPEG
		return "image/jpeg"
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x47, 0x49, 0x46, 0x38}): // GIF
		return "image/gif"
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x25, 0x50, 0x44, 0x46}): // PDF
		return "application/pdf"
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x50, 0x4B, 0x03, 0x04}): // ZIP
		return "application/zip"
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x52, 0x61, 0x72, 0x21}): // RAR
		return "application/x-rar-compressed"
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x1F, 0x8B, 0x08, 0x00}): // GZIP
		return "application/gzip"
	case len(data) >= 2 && bytes.Equal(data[:2], []byte{0x42, 0x4D}): // BMP
		return "image/bmp"
	default:
		return ""
	}
}

func mimeTypeText(mimeType []byte) bool {
	if bytes.HasPrefix(mimeType, []byte("text/")) {
		return true
	}

	if bytes.Equal(mimeType, []byte("TEXT")) ||
		bytes.Equal(mimeType, []byte("STRING")) ||
		bytes.Equal(mimeType, []byte("UTF8_STRING")) {
		return true
	}

	if bytes.Contains(mimeType, []byte("json")) ||
		bytes.HasSuffix(mimeType, []byte("script")) ||
		bytes.HasSuffix(mimeType, []byte("xml")) ||
		bytes.HasSuffix(mimeType, []byte("yaml")) ||
		bytes.HasSuffix(mimeType, []byte("csv")) ||
		bytes.HasSuffix(mimeType, []byte("ini")) {
		return true
	}

	return bytes.Contains(mimeType, []byte("application/vnd.ms-publisher")) ||
		bytes.HasSuffix(mimeType, []byte("pgp-keys"))
}
