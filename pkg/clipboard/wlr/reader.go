//go:build unix

package wlr

import (
	"strings"
	"sync/atomic"
	"time"

	"github.com/labi-le/belphegor/pkg/mime"
	"github.com/labi-le/belphegor/pkg/pipe/pipe"
	"github.com/rs/zerolog"
)

const (
	debounce = 200 * time.Millisecond
)

type ClipboardData struct {
	Data     []byte
	MimeType mime.Type
}

type reader struct {
	*preset
	logger zerolog.Logger

	currentOffer *controlOffer
	mimeTypes    []string
	dataChan     chan<- ClipboardData

	ignoreNextSelection atomic.Bool
	currentPipe         atomic.Pointer[pipe.Pipe]
}

func newReader(preset *preset, dataChan chan<- ClipboardData, log zerolog.Logger) *reader {
	return &reader{
		preset:   preset,
		dataChan: dataChan,
		logger:   log.With().Str("component", "reader").Logger(),
	}
}

func (r *reader) reset() {
	if old := r.currentPipe.Swap(nil); old != nil {
		_ = old.Close()
	}
}

func (r *reader) commit(p *pipe.Pipe) {
	r.currentPipe.Store(p)
}

func (r *reader) valid(p *pipe.Pipe) bool {
	return r.currentPipe.Load() == p
}

func (r *reader) release(p *pipe.Pipe) {
	r.currentPipe.CompareAndSwap(p, nil)
}

func (r *reader) IgnoreNextSelection() {
	r.ignoreNextSelection.Store(true)
	time.AfterFunc(debounce, func() {
		r.ignoreNextSelection.Store(false)
	})
}

func (r *reader) DataOffer(id *controlOffer) {
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

func (r *reader) Selection(offer *controlOffer) {
	r.reset()

	if offer == nil {
		r.logger.Trace().Msg("selection cleared (nil offer)")
		return
	}

	if r.ignoreNextSelection.Load() {
		return
	}

	selectedMime := r.selectBestMimeType()
	if selectedMime == "" {
		r.logger.Debug().
			Uint32("offer_id", offer.ID()).
			Strs("available_mimes", r.mimeTypes).
			Msg("no supported MIME type")
		return
	}

	r.logger.Trace().
		Uint32("offer_id", offer.ID()).
		Str("mime", selectedMime).
		Msg("selected MIME type")

	p, err := pipe.New()
	if err != nil {
		r.logger.Error().Err(err).Msg("failed to create pipe")
		return
	}

	offer.Receive(selectedMime, p.Fd())
	_ = p.Fd().Close()

	if err := r.client.RoundTrip(); err != nil {
		r.logger.Error().Err(err).Msg("round trip failed")
		_ = p.Close()
		return
	}

	r.commit(p)
	go r.readPipeData(selectedMime, p)
}

func (r *reader) selectBestMimeType() string {
	for _, availMime := range r.mimeTypes {
		if mime.IsSupported(availMime) {
			return availMime
		}
		if idx := strings.Index(availMime, ";"); idx != -1 {
			if mime.IsSupported(availMime[:idx]) {
				return availMime
			}
		}
	}
	return ""
}

func (r *reader) readPipeData(mimeType string, p *pipe.Pipe) {
	defer func() {
		r.release(p)
		_ = p.Close()
	}()

	r.logger.Trace().Msg("starting to read from pipe")

	data, err := pipe.FromPipe(p.ReadFd().Fd())
	if err != nil {
		if isExpectedError(err) {
			r.logger.Trace().Msg("read cancelled (pipe closed)")
		} else {
			r.logger.Error().Err(err).Msg("failed to read")
		}
		return
	}

	if !r.valid(p) {
		r.logger.Trace().Msg("dropping stale data")
		return
	}

	r.logger.Trace().Int("bytes", len(data)).Msg("read data")
	r.dataChan <- ClipboardData{
		Data:     data,
		MimeType: mime.AsType(mimeType),
	}
}

func isExpectedError(err error) bool {
	s := err.Error()
	return strings.Contains(s, "bad file descriptor") ||
		strings.Contains(s, "poll error") ||
		strings.Contains(s, "file already closed") ||
		strings.Contains(s, "use of closed network connection")
}

func (r *reader) Finished() {}

func (r *reader) PrimarySelection(*controlOffer) {}

func (r *reader) Offer(mimeType string) {
	r.mimeTypes = append(r.mimeTypes, mimeType)
}

func (r *reader) Close() error {
	r.reset()
	if r.currentOffer != nil {
		r.currentOffer.Destroy()
	}
	return nil
}
