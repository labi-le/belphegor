package wlr

import (
	"context"
	wl "deedles.dev/wl/client"
	"github.com/labi-le/belphegor/pkg/pipe"
	"github.com/rs/zerolog/log"
)

type ClipboardReader struct {
	*preset
	device *ZwlrDataControlDeviceV1
	Pipe   pipe.RWPipe
}

func NewClipboardReader(client *wl.Client, p pipe.RWPipe) *ClipboardReader {
	return &ClipboardReader{
		preset: &preset{client: client},
		Pipe:   p,
	}
}

func (s *ClipboardReader) Run(ctx context.Context) error {
	err := s.Setup()
	if err != nil {
		return err
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

func (d *deviceListener) Finished() {}

func (d *deviceListener) PrimarySelection(*ZwlrDataControlOfferV1) {}

type dataOfferListener ClipboardReader

func (d *dataOfferListener) Offer(string) {
	//log.Debug().Msgf("offer called with: %s", s)
}

func (s *ClipboardReader) Close() error {
	return s.client.Close()
}
