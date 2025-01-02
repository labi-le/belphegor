package wl

import (
	"context"
	wl "deedles.dev/wl/client"
	"errors"
	"fmt"
	"github.com/labi-le/belphegor/pkg/pipe"
	"github.com/rs/zerolog/log"
)

type state struct {
	client        *wl.Client
	registry      *wl.Registry
	seat          *wl.Seat
	deviceManager *ZwlrDataControlManagerV1
	display       *wl.Display
	pipe          pipe.Reusable
}

func (s *state) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-s.client.Events():
			if !ok {
				return
			}

			err := ev()
			if err != nil {
				log.Error().Msgf("event: %v", err)
			}
		}
	}
}

type displayListener state

func (lis *displayListener) Error(uint32, uint32, string) {}

func (lis *displayListener) DeleteId(id uint32) { lis.client.Delete(id) }

type registryListener state

func (lis *registryListener) GlobalRemove(uint32) {}

func (lis *registryListener) Global(name uint32, inter string, version uint32) {
	switch inter {
	case wl.SeatInterface:
		lis.seat = wl.BindSeat(lis.client, lis.registry, name, version)
	case ZwlrDataControlManagerV1Interface:
		lis.deviceManager = BindZwlrDataControlManagerV1(lis.client, lis.registry, name, version)
		log.Trace().Msgf("Bound data device manager: %v", lis.deviceManager)
	}
}

type deviceListener state

func (d *deviceListener) DataOffer(id *ZwlrDataControlOfferV1) {
	id.Listener = (*dataOfferListener)(d)
}

func (d *deviceListener) Selection(offer *ZwlrDataControlOfferV1) {
	if offer == nil {
		log.Trace().Msg("No selection data offer.")
		return
	}
	log.Trace().Msg("Received new selection offer.")

	mimeType := "text/plain"

	offer.Receive(mimeType, int(d.pipe.Fd()))
}

func (d *deviceListener) Finished() {
	log.Trace().Msg("sent eot")
}

func (d *deviceListener) PrimarySelection(*ZwlrDataControlOfferV1) {}

type dataOfferListener state

func (d *dataOfferListener) Offer(string) {}

func (s *state) init() error {
	client, err := wl.Dial()
	if err != nil {
		return fmt.Errorf("dial display: %w", err)
	}
	s.client = client

	s.display = client.Display()
	s.display.Listener = (*displayListener)(s)

	s.registry = s.display.GetRegistry()
	s.registry.Listener = (*registryListener)(s)

	err = s.client.RoundTrip()
	if err != nil {
		return fmt.Errorf("round trip: %w", err)
	}

	if s.seat == nil {
		return errors.New("no seat found")
	}

	device := s.deviceManager.GetDataDevice(s.seat)
	device.Listener = (*deviceListener)(s)

	return nil
}
