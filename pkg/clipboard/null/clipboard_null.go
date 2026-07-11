//go:build null

package null

import (
	"bytes"
	"context"
	"os"
	"time"

	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/mime"
	"github.com/rs/zerolog"
)

var _ eventful.Eventful = (*Clipboard)(nil)

// Headless driver knobs. With neither set the backend is an inert no-op
// (the historical behaviour of the `null` build). When set it becomes fully
// drivable through the filesystem, which is what the e2e tests rely on:
//   - IN  file: whenever its content changes it is surfaced as a local copy.
//   - OUT file: every clipboard write received from a peer is appended.
const (
	envIn        = "BELPHEGOR_HEADLESS_IN"
	envOut       = "BELPHEGOR_HEADLESS_OUT"
	pollInterval = 100 * time.Millisecond
)

// Clipboard is a display-less clipboard backend (built with -tags null).
type Clipboard struct {
	incoming chan []byte
	dedup    eventful.Deduplicator
	logger   zerolog.Logger
	inPath   string
	outPath  string
}

func New(logger zerolog.Logger, _ eventful.Options) *Clipboard {
	return &Clipboard{
		incoming: make(chan []byte, 16),
		logger:   logger.With().Str("component", "null").Logger(),
		inPath:   os.Getenv(envIn),
		outPath:  os.Getenv(envOut),
	}
}

// Watch emits every locally-originated copy as an Update until ctx is done.
func (n *Clipboard) Watch(ctx context.Context, upd chan<- eventful.Update) error {
	defer close(upd)

	if n.inPath != "" {
		go n.pollInput(ctx)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case data := <-n.incoming:
			h, isNew := n.dedup.Check(data)
			if !isNew {
				continue
			}
			upd <- eventful.Update{
				Data:     data,
				Size:     uint64(len(data)),
				MimeType: mime.From(data),
				Hash:     h,
			}
		}
	}
}

// Write records an incoming clipboard payload received from a peer. It marks
// the payload as seen (so it is not re-broadcast) and, when an OUT file is
// configured, appends it there so tests can observe injection.
func (n *Clipboard) Write(_ mime.Type, data []byte) (int, error) {
	n.dedup.Mark(data)

	if n.outPath != "" {
		if err := appendPayload(n.outPath, data); err != nil {
			return 0, err
		}
	}

	return len(data), nil
}

// pollInput watches the IN file and pushes new content as a local copy.
func (n *Clipboard) pollInput(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var last []byte
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			data, err := os.ReadFile(n.inPath)
			if err != nil {
				continue
			}
			data = bytes.TrimRight(data, "\r\n")
			if len(data) == 0 || bytes.Equal(data, last) {
				continue
			}
			last = data

			n.logger.Trace().Int("bytes", len(data)).Msg("headless copy detected")
			select {
			case n.incoming <- data:
			case <-ctx.Done():
				return
			}
		}
	}
}

func appendPayload(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if _, err := f.Write(data); err != nil {
		return err
	}
	_, err = f.Write([]byte{'\n'})
	return err
}
