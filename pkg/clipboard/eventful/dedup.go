package eventful

import (
	"sync/atomic"

	"github.com/cespare/xxhash"
)

type Deduplicator struct {
	lastHash atomic.Uint64
}

func (d *Deduplicator) Check(data []byte) (uint64, bool) {
	h := xxhash.Sum64(data)
	if h == d.lastHash.Load() {
		return h, false
	}
	d.lastHash.Store(h)
	return h, true
}

func (d *Deduplicator) Mark(data []byte) {
	d.lastHash.Store(xxhash.Sum64(data))
}
