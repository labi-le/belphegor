package channel

import (
	"sync"

	"github.com/labi-le/belphegor/internal/types/domain"
)

type fifo[K comparable, V any] struct {
	mu    sync.Mutex
	limit int
	order []K
	data  map[K]V
}

type history = fifo[uint64, domain.EventAnnounce]

func newHistory(limit int) *history {
	return &history{
		limit: limit,
		order: make([]uint64, 0, limit),
		data:  make(map[uint64]domain.EventAnnounce, limit),
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
