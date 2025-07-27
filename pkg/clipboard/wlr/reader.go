package wlr

import (
	"errors"
	"github.com/labi-le/belphegor/pkg/pipe/pipe"
	"github.com/rs/zerolog"
)

type reader struct {
	*preset
	pipe   pipe.RWPipe
	logger zerolog.Logger
}

func newReader(preset *preset, pipe pipe.RWPipe, log zerolog.Logger) *reader {
	return &reader{
		preset: preset,
		pipe:   pipe,
		logger: log.With().Str("component", "reader").Logger(),
	}
}

func (r *reader) DataOffer(id *ZwlrDataControlOfferV1) {
	if id == nil {
		return
	}
	id.Listener = r
}

func (r *reader) Selection(offer *ZwlrDataControlOfferV1) {
	if offer == nil {
		return
	}

	r.logger.Trace().Str("Selection", "received new selection offer").Send()

	offer.Receive("text/plain", r.pipe.Fd())
}

func (r *reader) Finished() {}

func (r *reader) PrimarySelection(*ZwlrDataControlOfferV1) {}

func (r *reader) Offer(string) {
	//log.Debug().Msgf("offer called with: %s", s)
}

func (r *reader) Close() error {
	return errors.Join(r.client.Close(), r.pipe.Close())
}
