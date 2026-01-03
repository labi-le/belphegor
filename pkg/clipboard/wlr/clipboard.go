// FILE: pkg/clipboard/wlr/clipboard.go

//go:build unix

package wlr

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	wl "deedles.dev/wl/client"
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/rs/zerolog"
)

var Supported = (func() bool {
	_, exist1 := os.LookupEnv("WAYLAND_DISPLAY")
	_, exist2 := os.LookupEnv("WAYLAND_SOCKET")
	return exist1 || exist2
})()

type Clipboard struct {
	reader   *reader
	writer   *writer
	preset   *preset
	logger   zerolog.Logger
	dataChan chan ClipboardData
	closed   atomic.Bool
	mu       sync.Mutex
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

	dataChan := make(chan ClipboardData, 1)

	wlr := &Clipboard{
		preset:   preset,
		logger:   log.With().Str("component", "wlr").Logger(),
		dataChan: dataChan,
	}

	wlr.reader = newReader(preset, dataChan, log)
	wlr.writer = newWriter(preset, wlr.reader, log)

	return wlr
}

func (w *Clipboard) Watch(ctx context.Context, update chan<- eventful.Update) error {
	defer close(update)
	log := w.logger.With().Str("op", "wlr.Watch").Logger()

	errCh := make(chan error, 1)
	go func() {
		errCh <- w.Run(ctx)
	}()

	for {
		select {
		case <-ctx.Done():
			return nil

		case runErr := <-errCh:
			if runErr != nil {
				return fmt.Errorf("wlr.Watch: %w", runErr)
			}
			return nil

		case clipData, ok := <-w.dataChan:
			if !ok {
				log.Trace().Msg("*wlr.dataChan closed")
				return nil
			}
			if len(clipData.Data) > 0 {
				update <- eventful.Update{Data: clipData.Data, MimeType: clipData.MimeType}
			}
		}
	}
}

func (w *Clipboard) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	log := w.logger.With().Str("op", "wlr.Write").Logger()

	if w.closed.Load() {
		log.Warn().
			Bool("closed", true).
			Msg("wlr is closed, ignoring write")
		return 0, errors.New("clipboard is closed")
	}

	return w.writer.Write(data)
}

func (w *Clipboard) Run(ctx context.Context) error {
	log := w.logger.With().Str("op", "wlr.Run").Logger()

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
			return ctx.Err()
		case ev, ok := <-w.preset.client.Events():
			if !ok {
				return nil
			}
			w.mu.Lock()
			err := ev()
			w.mu.Unlock()
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

	w.mu.Lock()
	defer w.mu.Unlock()

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

	close(w.dataChan)

	return nil
}
