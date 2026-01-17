package eventful

import (
	"sync/atomic"

	"github.com/cespare/xxhash"
)

var Hasher = xxhash.New

type Deduplicator struct {
	lastHash atomic.Uint64
}

func (d *Deduplicator) Check(data []byte) (hash uint64, isNew bool) {
	h := d.Hash(data)
	if h == d.lastHash.Load() {
		return h, false
	}
	d.lastHash.Store(h)
	return h, true
}

func (d *Deduplicator) Mark(data []byte) {
	d.lastHash.Store(d.Hash(data))
}

func (d *Deduplicator) Hash(data []byte) uint64 {
	return xxhash.Sum64(data)
}
