package wlr

import (
	"context"
	wl "deedles.dev/wl/client"
	"errors"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/pipe/pipe"
	"github.com/labi-le/belphegor/pkg/pool/byteslice"
	"github.com/rs/zerolog"
	"io"
	"os"
	"syscall"
)

var Supported = (func() bool {
	_, exist1 := os.LookupEnv("WAYLAND_DISPLAY")
	_, exist2 := os.LookupEnv("WAYLAND_SOCKET")
	return exist1 || exist2
})()

type Wlr struct {
	reader *reader
	writer *writer
	preset *preset
	logger zerolog.Logger
}

func Must(log zerolog.Logger) *Wlr {
	r, err := wl.Dial()
	if err != nil {
		panic(err)
	}
	return New(r, pipe.MustNonBlock(log), log)
}

func New(client *wl.Client, p pipe.RWPipe, log zerolog.Logger) *Wlr {
	preset := newPreset(client, log)
	return &Wlr{
		reader: newReader(preset, p, log),
		writer: newWriter(preset, log),
		preset: preset,
		logger: log.With().Str("component", "wlr").Logger(),
	}
}

func (w *Wlr) Watch(ctx context.Context, update chan<- clipboard.Update) {
	go func() {
		err := w.Run(ctx)
		if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
			w.logger.Error().Err(err).Msg("wlr.Run exited with error")
		}
		close(update)
	}()

	go func() {
		buffer := byteslice.Get(65536) // 64KB buffer
		defer byteslice.Put(buffer)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := syscall.Read(int(w.reader.pipe.ReadFd().Fd()), buffer)
				if err != nil {
					if err == io.EOF || errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.EBADF) {
						w.logger.Trace().Msg("Pipe closed, exiting watch loop.")
						return
					}
					if errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EWOULDBLOCK) {
						continue
					}

					update <- clipboard.Update{Data: nil, Err: err}
					continue
				}

				if n > 0 {
					dataCopy := make([]byte, n)
					copy(dataCopy, buffer[:n])
					update <- clipboard.Update{Data: dataCopy, Err: nil}
				}
			}
		}
	}()

	<-ctx.Done()
}

func (w *Wlr) Write(data []byte) (int, error) {
	return w.writer.Write(data)
}

func (w *Wlr) Run(ctx context.Context) error {
	err := w.preset.Setup()
	if err != nil {
		return err
	}

	w.preset.device.Listener = w.reader
	w.preset.device.OnDelete = func() {
		_ = w.preset.client.RoundTrip()
		w.logger.Trace().Uint32("device.OnDelete free", w.preset.device.ID()).Send()
	}
	//
	//w.device = w.deviceManager.GetDataDevice(w.seat)
	//w.device.Listener = (*deviceListener2)(w)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-w.preset.client.Events():
			if !ok {
				return nil
			}
			err := ev()
			if err != nil {
				w.logger.Err(err).Send()
				return err
			}
		}
	}
}

func (w *Wlr) Close() error {
	if err := w.writer.Close(); err != nil {
		return err
	}

	return w.reader.Close()
}
