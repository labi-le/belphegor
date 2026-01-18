package channel

import (
	"sync"

	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/id"
)

type fifo[K comparable, V any] struct {
	mu    sync.Mutex
	limit int
	order []K
	data  map[K]V
}

type (
	announceHistory    = fifo[uint64, domain.EventAnnounce]
	servedFilesHistory = fifo[id.Unique, domain.EventMessage]
)

func newHistory(limit int) *announceHistory {
	return &announceHistory{
		limit: limit,
		order: make([]uint64, 0, limit),
		data:  make(map[uint64]domain.EventAnnounce, limit),
	}
}
func newServedFilesHistory(limit int) *servedFilesHistory {
	return &servedFilesHistory{
		limit: historySize,
		order: make([]id.Unique, 0, limit),
		data:  make(map[id.Unique]domain.EventMessage, limit),
	}
}

func (h *fifo[K, V]) Add(key K, value V) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.data[key]; ok {
		return false
	}

	if len(h.order) >= h.limit {
		pop := h.order[0]
		h.order = h.order[1:]
		delete(h.data, pop)
	}

	h.order = append(h.order, key)
	h.data[key] = value
	return true
}

func (h *fifo[K, V]) Get(key K) (V, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	val, ok := h.data[key]
	return val, ok
}
