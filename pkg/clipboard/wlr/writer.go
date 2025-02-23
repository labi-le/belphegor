package wlr

import (
	"bytes"
	"errors"
	"github.com/rs/zerolog"
	"syscall"
)

type writer struct {
	*preset
	logger zerolog.Logger
}

func newWriter(preset *preset, log zerolog.Logger) *writer {
	return &writer{
		preset: preset,
		logger: log.With().Str("component", "writer").Logger(),
	}
}

//func ClipboardSet2(data []byte, cmd *exec.Cmd) error {
//	var (
//		in  io.WriteCloser
//		err error
//	)
//
//	if in, err = cmd.StdinPipe(); err != nil {
//		return err
//	}
//
//	if err = cmd.Start(); err != nil {
//		return err
//	}
//
//	if _, err = in.Write(data); err != nil {
//		return err
//	}
//
//	if err = in.Close(); err != nil {
//		return err
//	}
//
//	return cmd.Wait()
//}

func (w *writer) Write(p []byte) (n int, err error) {
	// plan b
	//return len(p), ClipboardSet2(p,
	//	exec.Command("wl-copy"),
	//)
	//
	if w.deviceManager == nil {
		return 0, errors.New("data control manager not initialized")
	}

	if len(p) == 0 {
		return 0, nil
	}

	activeSource := w.deviceManager.CreateDataSource()

	typ := mimeType(p)
	if typ == "" || mimeTypeText(p) {
		activeSource.Offer("TEXT")
		activeSource.Offer("STRING")
		activeSource.Offer("UTF8_STRING")
		w.logger.Trace().Str("active source", "generic string").Send()
	} else {
		activeSource.Offer(typ)
		w.logger.Trace().Str("active source", typ).Send()
	}

	activeSource.Listener = newDevListener{p, activeSource, w.logger}
	activeSource.OnDelete = func() {
		w.logger.Trace().Str("writer.Write", "Write").Msgf("free active source: %d", activeSource.ID())
		_ = w.client.RoundTrip()
	}
	w.device.SetSelection(activeSource)
	return len(p), nil
}

func (w *writer) Close() error {
	return w.client.Close()
}

type newDevListener struct {
	data   []byte
	source *ZwlrDataControlSourceV1
	logger zerolog.Logger
}

func (n newDevListener) Send(mime string, fd int) {
	defer n.source.Delete()

	ctxLog := n.logger.With().Str("op", "Send").Logger()
	ctxLog.Trace().Msgf("write data to fd: %d", fd)

	written, err := syscall.Write(fd, n.data)
	if err != nil {
		ctxLog.Error().Msgf("failed to write clipboard data: %v", err)
		return
	}
	if err := syscall.Close(fd); err != nil {
		ctxLog.Error().Msgf("failed to close fd: %v", err)
		return
	}
	ctxLog.Trace().Msgf("send n: %d, %s fd: %d", written, mime, fd)
}

func (n newDevListener) Cancelled() {
	if n.source != nil {
		n.source.Delete()
		n.source.Destroy()
		n.logger.Trace().Str("Cancelled", "called cancelled")
	}
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
