//go:build unix

package wlr

import (
	"errors"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cespare/xxhash"
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/mime"
	"github.com/labi-le/belphegor/pkg/pipe/pipe"
	"github.com/rs/zerolog"
)

const (
	debounce = 200 * time.Millisecond
)

type reader struct {
	*preset
	logger zerolog.Logger

	currentOffer *controlOffer
	mimeTypes    []string
	dataChan     chan<- eventful.Update

	lastHash atomic.Uint64
	barrier  atomic.Int64
}

func newReader(preset *preset, dataChan chan<- eventful.Update, log zerolog.Logger) *reader {
	return &reader{
		preset:   preset,
		dataChan: dataChan,
		logger:   log.With().Str("component", "reader").Logger(),
	}
}

// Suppress sets a barrier to the future, ignoring events for the duration of the debounce
// for writer
func (r *reader) Suppress() {
	deadline := time.Now().Add(debounce).UnixNano()
	r.barrier.Store(deadline)
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
	if !r.allowed() {
		return
	}

	if offer == nil {
		r.logger.Trace().Msg("selection cleared (nil offer)")
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

	p, err := pipe.New()
	if err != nil {
		r.logger.Error().Err(err).Msg("failed to create pipe")
		return
	}

	offer.Receive(selectedMime, p.Fd())

	go r.readPipeData(selectedMime, p)
}

// allowed sliding window debounce
func (r *reader) allowed() bool {
	now := time.Now().UnixNano()
	deadline := r.barrier.Load()
	newDeadline := now + int64(debounce)

	if now < deadline {
		r.barrier.Store(newDeadline)
		//r.logger.Trace().Msg("debounce: selection ignored")
		return false
	}

	r.barrier.Store(newDeadline)
	return true
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
	defer func() { _ = p.Close() }()

	timer := time.AfterFunc(debounce, func() { _ = p.Close() })
	defer timer.Stop()

	data, err := pipe.FromPipe2(p.ReadFd())
	if err != nil && !errors.Is(err, os.ErrClosed) {
		if isExpectedError(err) {
			r.logger.Trace().Msg("read cancelled (pipe closed)")
		} else {
			r.logger.Error().Err(err).Msg("failed to read")
		}
		return
	}

	if len(data) == 0 {
		return
	}

	if !r.dedup(data) {
		return
	}

	r.dataChan <- eventful.Update{
		Data:     data,
		MimeType: mime.AsType(mimeType),
		Hash:     r.lastHash.Load(),
	}
}

func (r *reader) dedup(data []byte) bool {
	dataHash := xxhash.Sum64(data)

	if dataHash == r.lastHash.Load() {
		return false
	}

	r.lastHash.Store(dataHash)
	return true
}

func isExpectedError(err error) bool {
	s := err.Error()
	return strings.Contains(s, "bad file descriptor") ||
		strings.Contains(s, "poll error") ||
		strings.Contains(s, "use of closed network connection")
}

func (r *reader) Finished() {}

func (r *reader) PrimarySelection(*controlOffer) {}

func (r *reader) Offer(mimeType string) {
	r.mimeTypes = append(r.mimeTypes, mimeType)
}

func (r *reader) Close() error {
	if r.currentOffer != nil {
		r.currentOffer.Destroy()
	}
	return nil
}
