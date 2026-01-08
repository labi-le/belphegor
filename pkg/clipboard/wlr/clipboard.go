//go:build unix

package wlr

import (
	"context"
	"errors"
	"os"
	"sync/atomic"

	wl "deedles.dev/wl/client"
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/mime"
	"github.com/rs/zerolog"
)

var _ eventful.Eventful = (*Clipboard)(nil)

var Supported = (func() bool {
	_, exist1 := os.LookupEnv("WAYLAND_DISPLAY")
	_, exist2 := os.LookupEnv("WAYLAND_SOCKET")
	return exist1 || exist2
})()

type Clipboard struct {
	reader *reader
	writer *writer
	preset *preset
	logger zerolog.Logger
	closed atomic.Bool
}

func Must(log zerolog.Logger) *Clipboard {
	client, err := wl.Dial()
	if err != nil {
		panic(err)
	}
	return New(client, log)
}

func New(client *wl.Client, log zerolog.Logger) *Clipboard {
	preset := newPreset(client, log)

	wlr := &Clipboard{
		preset: preset,
		logger: log.With().Str("component", "wlr").Logger(),
	}

	return wlr
}

func (w *Clipboard) Watch(ctx context.Context, upd chan<- eventful.Update) error {
	defer close(upd)
	log := w.logger.With().Str("op", "wlr.Watch").Logger()

	w.reader = newReader(w.preset, upd, log)
	w.writer = newWriter(w.preset, w.reader, log)

	return w.run(ctx)
}

func (w *Clipboard) Write(t mime.Type, src []byte) (int, error) {
	log := w.logger.With().Str("op", "wlr.Write").Logger()

	if w.closed.Load() {
		log.Warn().
			Bool("closed", true).
			Msg("wlr is closed, ignoring write")
		return 0, errors.New("clipboard is closed")
	}

	return w.writer.Write(t, src)
}

func (w *Clipboard) run(ctx context.Context) error {
	log := w.logger.With().Str("op", "wlr.run").Logger()

	err := w.preset.Setup()
	if err != nil {
		return err
	}

	w.preset.device.Listener = w.reader

	w.preset.device.OnDelete = func() {
		log.Trace().
			Uint32("device_id", w.preset.device.ID()).
			Msg("device deleted")
	}

	for {
		select {
		case <-ctx.Done():
			return w.preset.client.Close()
		case ev, ok := <-w.preset.client.Events():
			if !ok {
				return nil
			}
			err := ev()
			if err != nil {
				log.Error().
					Err(err).
					Msg("event processing error")
				return err
			}
		}
	}
}

func (w *Clipboard) Close() error {
	log := w.logger.With().Str("op", "wlr.Close").Logger()

	log.Debug().
		Msg("closing wlr clipboard")

	w.closed.Store(true)

	if err := w.writer.Close(); err != nil {
		log.Error().
			Str("closer", "writer").
			Err(err).
			Msg("failed to close writer")
	}

	if err := w.reader.Close(); err != nil {
		log.Error().
			Str("closer", "reader").
			Err(err).
			Msg("failed to close reader")
	}

	if err := w.preset.client.Close(); err != nil {
		log.Error().
			Str("closer", "client").
			Err(err).
			Msg("failed to close client")
		return err
	}

	return nil
}
