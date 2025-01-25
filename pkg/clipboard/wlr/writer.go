package wlr

import (
	"context"
	wl "deedles.dev/wl/client"
	"errors"
	"github.com/rs/zerolog/log"
	"syscall"
)

type ClipboardWriter struct {
	*preset
	device *ZwlrDataControlDeviceV1
	data   []byte
}

func NewClipboardWriter(client *wl.Client) *ClipboardWriter {
	return &ClipboardWriter{
		preset: &preset{client: client},
	}
}

func (w *ClipboardWriter) Write(p []byte) (n int, err error) {
	if w.deviceManager == nil {
		return 0, errors.New("data control manager not initialized")
	}

	if len(p) == 0 {
		return 0, nil
	}

	activeSource := w.deviceManager.CreateDataSource()
	activeSource.Offer("text/plain;charset=utf-8")
	activeSource.Listener = newDevListener{p, activeSource}
	activeSource.OnDelete = func() {
		w.client.RoundTrip()
	}
	w.data = p
	w.device.SetSelection(activeSource)
	return len(p), nil
}

func (w *ClipboardWriter) Close() error {
	return w.client.Close()
}

func (w *ClipboardWriter) Run(ctx context.Context) error {
	err := w.Setup()
	if err != nil {
		return err
	}
	w.device = w.deviceManager.GetDataDevice(w.seat)
	w.device.Listener = (*deviceListener2)(w)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-w.client.Events():
			if !ok {
				return nil
			}
			err := ev()
			if err != nil {
				log.Error().Msgf("event: %v", err)
				return err
			}
		}
	}
}

type deviceListener2 ClipboardWriter

func (d *deviceListener2) DataOffer(id *ZwlrDataControlOfferV1) {
	log.Trace().Msgf("data offer %s", id)
}

func (d *deviceListener2) Selection(offer *ZwlrDataControlOfferV1) {
	if offer != nil {
		offer.Destroy()
	}
}

func (d *deviceListener2) Finished() {}

func (d *deviceListener2) PrimarySelection(*ZwlrDataControlOfferV1) {}

type newDevListener struct {
	data   []byte
	source *ZwlrDataControlSourceV1
}

func (n newDevListener) Send(mime string, fd int) {
	defer n.source.Delete()
	written, err := syscall.Write(fd, n.data)
	if err != nil {
		log.Error().Msgf("failed to write clipboard data: %v", err)
		return
	}
	if err := syscall.Close(fd); err != nil {
		log.Error().Msgf("failed to close fd: %v", err)
		return
	}
	log.Debug().Msgf("send n: %d, %s fd: %d", written, mime, fd)
}

func (n newDevListener) Cancelled() {
	if n.source != nil {
		n.source.Destroy()
	}
	log.Trace().Msg("called cancelled")
}
