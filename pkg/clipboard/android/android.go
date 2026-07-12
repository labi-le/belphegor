// Package android implements the eventful.Eventful clipboard backend for
// hosts that cannot be watched from Go directly (Android, iOS). Instead of
// talking to a display server, it bridges to the host platform:
//
//   - Local copies are pushed in from the host via PushLocalCopy and surfaced
//     to the belphegor core through Watch.
//   - Remote payloads handed to Write are forwarded to a host-provided Writer
//     (e.g. the Android ClipboardManager, reached over the gomobile boundary).
//
// The package is deliberately free of platform-specific imports so it compiles
// and unit-tests on every GOOS; the platform seam is the injected Writer.
package android

import (
	"context"
	"fmt"
	"strings"

	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/mime"
	"github.com/rs/zerolog"
)

// Writer sinks a remote clipboard payload into the host clipboard. mimeType is
// a mime.Type label ("text", "image", "path"). Implemented on the host side;
// under gomobile this becomes a Java/Kotlin interface.
type Writer interface {
	Write(mimeType string, data []byte) error
}

var _ eventful.Eventful = (*Clipboard)(nil)

// localBuffer bounds how many un-drained local copies we hold before dropping.
const localBuffer = 16

// Clipboard bridges the belphegor core to a host-provided clipboard.
type Clipboard struct {
	incoming chan eventful.Update
	dedup    eventful.Deduplicator
	writer   Writer
	logger   zerolog.Logger
}

// New builds a bridge. writer may be nil, in which case remote payloads are
// logged and dropped instead of being injected (used by the generic
// clipboard.New dispatch; the real app always injects a host writer).
func New(logger zerolog.Logger, _ eventful.Options, writer Writer) *Clipboard {
	return &Clipboard{
		incoming: make(chan eventful.Update, localBuffer),
		writer:   writer,
		logger:   logger.With().Str("component", "android").Logger(),
	}
}

// Watch forwards local copies to upd until ctx is done. It honours the
// Eventful contract by closing upd on return.
func (c *Clipboard) Watch(ctx context.Context, upd chan<- eventful.Update) error {
	defer close(upd)
	for {
		select {
		case <-ctx.Done():
			return nil
		case u := <-c.incoming:
			select {
			case upd <- u:
			case <-ctx.Done():
				return nil
			}
		}
	}
}

// Write injects a remote payload into the host clipboard. The payload is
// marked as seen first, so the clipboard change it triggers on the host is not
// picked up by PushLocalCopy and re-broadcast (loop protection).
func (c *Clipboard) Write(t mime.Type, data []byte) (int, error) {
	c.dedup.Mark(data)

	if c.writer == nil {
		c.logger.Warn().Msg("no clipboard writer configured; dropping remote payload")
		return len(data), nil
	}

	if err := c.writer.Write(t.String(), data); err != nil {
		return 0, fmt.Errorf("android clipboard write: %w", err)
	}
	return len(data), nil
}

// PushLocalCopy surfaces a clipboard change that originated on this device so
// the core can broadcast it. mimeHint is the platform content-type (e.g. an
// Android ClipDescription MIME: "text/plain", "image/png", "text/uri-list"),
// or "" to classify the payload by its bytes. Payloads identical to the last
// seen one are dropped (dedup / loop protection), matching the desktop
// backends.
func (c *Clipboard) PushLocalCopy(mimeHint string, data []byte) {
	if len(data) == 0 {
		return
	}

	hash, isNew := c.dedup.Check(data)
	if !isNew {
		return
	}

	kind := mime.From(data)
	// The Android ClipDescription MIME is a strong hint for images (raw bytes
	// may otherwise classify ambiguously); honour it without a core mime helper.
	if strings.HasPrefix(mimeHint, "image/") {
		kind = mime.TypeImage
	}

	upd := eventful.Update{
		Data:     data,
		Size:     uint64(len(data)),
		MimeType: kind,
		Hash:     hash,
	}

	select {
	case c.incoming <- upd:
	default:
		c.logger.Warn().Msg("local copy dropped: buffer full")
	}
}
