package wlr

import (
	"strings"
	"sync"
	"time"

	"github.com/labi-le/belphegor/pkg/mime"
	"github.com/labi-le/belphegor/pkg/pipe/pipe"
	"github.com/rs/zerolog"
)

type ClipboardData struct {
	Data     []byte
	MimeType string
}

type reader struct {
	*preset
	logger zerolog.Logger

	currentOffer *ZwlrDataControlOfferV1
	mimeTypes    []string
	dataChan     chan<- ClipboardData

	mu                  sync.Mutex
	ignoreNextSelection bool
}

func newReader(preset *preset, dataChan chan<- ClipboardData, log zerolog.Logger) *reader {
	return &reader{
		preset:   preset,
		dataChan: dataChan,
		logger:   log.With().Str("component", "reader").Logger(),
	}
}

func (r *reader) IgnoreNextSelection() {
	r.mu.Lock()
	r.ignoreNextSelection = true
	r.mu.Unlock()

	time.AfterFunc(200*time.Millisecond, func() {
		r.mu.Lock()
		r.ignoreNextSelection = false
		r.mu.Unlock()
	})
}

func (r *reader) DataOffer(id *ZwlrDataControlOfferV1) {
	if id == nil {
		return
	}

	if r.currentOffer != nil {
		r.currentOffer.Destroy()
	}

	r.currentOffer = id
	r.mimeTypes = nil
	id.Listener = r
}

func (r *reader) Selection(offer *ZwlrDataControlOfferV1) {
	if offer == nil {
		r.logger.Trace().
			Bool("offer_nil", true).
			Msg("selection cleared (nil offer)")
		return
	}

	r.mu.Lock()
	shouldIgnore := r.ignoreNextSelection
	r.mu.Unlock()

	if shouldIgnore {
		return
	}

	r.logger.Trace().
		Uint32("offer_id", offer.ID()).
		Strs("available_mimes", r.mimeTypes).
		Msg("selection received")

	selectedMime := r.selectBestMimeType()
	if selectedMime == "" {
		r.logger.Debug().
			Uint32("offer_id", offer.ID()).
			Strs("available_mimes", r.mimeTypes).
			Msg("no supported MIME type available")
		return
	}

	r.logger.Trace().
		Uint32("offer_id", offer.ID()).
		Str("mime", selectedMime).
		Msg("selected MIME type")

	p, err := pipe.New()
	if err != nil {
		r.logger.Error().
			Uint32("offer_id", offer.ID()).
			Str("mime", selectedMime).
			Err(err).
			Msg("failed to create pipe")
		return
	}

	offer.Receive(selectedMime, p.Fd())
	_ = p.Fd().Close()

	if err := r.client.RoundTrip(); err != nil {
		r.logger.Error().
			Uint32("offer_id", offer.ID()).
			Str("mime", selectedMime).
			Err(err).
			Msg("round trip failed")
	}

	go r.readPipeData(selectedMime, p)
}

func (r *reader) selectBestMimeType() string {
	for _, availMime := range r.mimeTypes {
		if mime.IsSupported(availMime) {
			return availMime
		}

		if idx := strings.Index(availMime, ";"); idx != -1 {
			base := availMime[:idx]
			if mime.IsSupported(base) {
				return availMime
			}
		}
	}

	return ""
}

func (r *reader) readPipeData(mimeType string, p pipe.RWPipe) {
	defer func() {
		if p != nil {
			_ = p.Close()
		}
	}()

	readFd := p.ReadFd().Fd()
	r.logger.Trace().
		Int("fd", int(readFd)).
		Str("mime", mimeType).
		Msg("starting to read from pipe")

	type result struct {
		data []byte
		err  error
	}

	resultChan := make(chan result, 1)

	go func() {
		data, err := pipe.FromPipe(readFd)
		resultChan <- result{data: data, err: err}
	}()

	select {
	case res := <-resultChan:
		if res.err != nil {
			errStr := res.err.Error()
			if strings.Contains(errStr, "bad file descriptor") ||
				strings.Contains(errStr, "poll error") {
				r.logger.Trace().
					Int("fd", int(readFd)).
					Str("mime", mimeType).
					Str("reason", errStr).
					Msg("offer was cancelled")
			} else {
				r.logger.Error().
					Int("fd", int(readFd)).
					Str("mime", mimeType).
					Err(res.err).
					Msg("failed to read from pipe")
			}
			return
		}

		if len(res.data) > 0 {
			r.logger.Trace().
				Int("fd", int(readFd)).
				Int("bytes_read", len(res.data)).
				Str("mime", mimeType).
				Msg("read data from pipe")

			r.dataChan <- ClipboardData{
				Data:     res.data,
				MimeType: mimeType,
			}
		}
	}
}

func (r *reader) Finished() {}

func (r *reader) PrimarySelection(*ZwlrDataControlOfferV1) {}

func (r *reader) Offer(mimeType string) {
	r.mimeTypes = append(r.mimeTypes, mimeType)
}

func (r *reader) Close() error {
	if r.currentOffer != nil {
		r.currentOffer.Destroy()
	}

	return nil
}
