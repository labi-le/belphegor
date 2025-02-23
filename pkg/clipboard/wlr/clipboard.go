package wlr

import (
	"context"
	wl "deedles.dev/wl/client"
	"github.com/labi-le/belphegor/pkg/clipboard/api"
	"github.com/labi-le/belphegor/pkg/pipe"
	"github.com/rs/zerolog"
	"os"
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

func (w *Wlr) Watch(ctx context.Context, update chan<- api.Update) {
	go func() {
		<-ctx.Done()
		close(update)
	}()

	for {
		select {
		case <-ctx.Done():
			update <- api.Update{Data: []byte{}, Err: ctx.Err()}
			return
		default:
			buf, err := pipe.FromPipe(w.reader.pipe.ReadFd())
			if err != nil {
				continue
			}

			update <- api.Update{Data: buf, Err: err}
		}
	}

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
