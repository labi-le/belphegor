package channel

import (
	"bytes"
	"slices"
	"sync"
	"time"

	"github.com/labi-le/belphegor/internal/types/domain"
)

var batchSeparator = []byte("\n")

const threshold = 3 * time.Minute

type batchState struct {
	total uint32
	items map[domain.MessageID][]byte
	timer *time.Timer
}

type BatchCollector struct {
	mu      sync.Mutex
	batches map[domain.MessageID]*batchState
}

func NewBatchCollector() *BatchCollector {
	return &BatchCollector{
		batches: make(map[domain.MessageID]*batchState),
	}
}

func (c *BatchCollector) Add(msg domain.Message) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bID := msg.BatchID
	state, exists := c.batches[bID]
	if !exists {
		state = &batchState{
			total: msg.BatchTotal,
			items: make(map[domain.MessageID][]byte),
		}
		state.timer = time.AfterFunc(threshold, func() {
			c.mu.Lock()
			defer c.mu.Unlock()
			delete(c.batches, bID)
		})
		c.batches[bID] = state
	}

	state.items[msg.ID] = msg.Data

	if uint32(len(state.items)) == state.total {
		state.timer.Stop()
		delete(c.batches, bID)
		return c.join(state.items), true
	}

	return nil, false
}

func (c *BatchCollector) join(items map[domain.MessageID][]byte) []byte {
	parts := make([][]byte, 0, len(items))
	for _, data := range items {
		if len(data) > 0 {
			parts = append(parts, data)
		}
	}
	slices.SortFunc(parts, bytes.Compare)
	return bytes.Join(parts, batchSeparator)
}
