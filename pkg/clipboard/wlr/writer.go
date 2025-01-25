package wlr

import (
	"context"
	wl "deedles.dev/wl/client"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"syscall"
)

type ClipboardWriter struct {
	client        *wl.Client
	registry      *wl.Registry
	seat          *wl.Seat
	display       *wl.Display
	deviceManager *ZwlrDataControlManagerV1
	device        *ZwlrDataControlDeviceV1
	data          []byte
}

func NewClipboardWriter(client *wl.Client) *ClipboardWriter {
	return &ClipboardWriter{client: client}
}

func (w *ClipboardWriter) Write(p []byte) (n int, err error) {
	if w.deviceManager == nil {
		return 0, errors.New("data control manager not initialized")
	}

	activeSource := w.deviceManager.CreateDataSource()
	//activeSource.Offer("text/plain")
	activeSource.Offer("text/plain;charset=utf-8")
	//activeSource.Offer("UTF8_STRING")
	//activeSource.Offer("TEXT")
	//activeSource.Offer("STRING")

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

func (w *ClipboardWriter) GlobalRemove(uint32) {}

func (w *ClipboardWriter) Global(name uint32, inter string, version uint32) {
	switch inter {
	case wl.SeatInterface:
		w.seat = wl.BindSeat(w.client, w.registry, name, version)
	case ZwlrDataControlManagerV1Interface:
		w.deviceManager = BindZwlrDataControlManagerV1(w.client, w.registry, name, version)
		log.Trace().Msgf("bound data device manager: %v", w.deviceManager)
	}
}

func (w *ClipboardWriter) Run(ctx context.Context) error {
	w.display = w.client.Display()
	w.registry = w.display.GetRegistry()
	w.registry.Listener = w

	if err := w.client.RoundTrip(); err != nil {
		return fmt.Errorf("round trip: %w", err)
	}

	if w.seat == nil {
		return errors.New("no seat found")
	}

	w.device = w.deviceManager.GetDataDevice(w.seat)
	w.device.Listener = (*deviceListener2)(w)

	go func() {
		<-ctx.Done()
		if w.client != nil {
			w.client.Close()
		}
	}()
	for ev := range w.client.Events() {
		if err := ev(); err != nil {
			log.Error().Msgf("event: %v", err)
			return err
		}
	}

	return nil
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
