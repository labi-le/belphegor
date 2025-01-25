package wlr

import (
	"context"
	wl "deedles.dev/wl/client"
	"errors"
	"fmt"
	"github.com/labi-le/belphegor/pkg/pipe"
	"github.com/rs/zerolog/log"
)

type ClipboardReader struct {
	client        *wl.Client
	registry      *wl.Registry
	seat          *wl.Seat
	deviceManager *ZwlrDataControlManagerV1
	device        *ZwlrDataControlDeviceV1
	display       *wl.Display
	Pipe          pipe.RWPipe
}

func NewClipboardReader(client *wl.Client, p pipe.RWPipe) *ClipboardReader {
	return &ClipboardReader{client: client, Pipe: p}
}

func (s *ClipboardReader) GlobalRemove(uint32) {}

func (s *ClipboardReader) Global(name uint32, inter string, version uint32) {
	switch inter {
	case wl.SeatInterface:
		s.seat = wl.BindSeat(s.client, s.registry, name, version)
	case ZwlrDataControlManagerV1Interface:
		s.deviceManager = BindZwlrDataControlManagerV1(s.client, s.registry, name, version)
		log.Trace().Msgf("bound data device manager: %v", s.deviceManager)
	}
}

func (s *ClipboardReader) Run(ctx context.Context) error {
	s.display = s.client.Display()

	s.registry = s.display.GetRegistry()
	s.registry.Listener = s

	err := s.client.RoundTrip()
	if err != nil {
		return fmt.Errorf("round trip: %w", err)
	}

	if s.seat == nil {
		return errors.New("no seat found")
	}

	s.device = s.deviceManager.GetDataDevice(s.seat)
	s.device.Listener = (*deviceListener)(s)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-s.client.Events():
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

type deviceListener ClipboardReader

func (d *deviceListener) DataOffer(id *ZwlrDataControlOfferV1) {
	if id == nil {
		return
	}
	id.Listener = (*dataOfferListener)(d)
}

func (d *deviceListener) Selection(offer *ZwlrDataControlOfferV1) {
	if offer == nil {
		return
	}

	log.Trace().Msg("received new selection offer")
	offer.Receive("text/plain", int(d.Pipe.Fd()))
}

func (d *deviceListener) Finished() {
	log.Trace().Msg("sent eot")
}

func (d *deviceListener) PrimarySelection(*ZwlrDataControlOfferV1) {}

type dataOfferListener ClipboardReader

func (d *dataOfferListener) Offer(s string) {
	//log.Debug().Msgf("offer called with: %s", s)
}

func (s *ClipboardReader) Close() error {
	return s.client.Close()
}
