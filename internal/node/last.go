package node

import (
	"github.com/labi-le/belphegor/internal/types/domain"
	"sync"
)

// LastMessage which is stored in Node and serves to identify duplicate messages
type LastMessage struct {
	*domain.Message
	mu     sync.Mutex
	Update chan *domain.Message
}

func NewLastMessage() *LastMessage {
	return &LastMessage{Update: make(chan *domain.Message)}
}

func (m *LastMessage) Get() *domain.Message {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Message
}

// ListenUpdates updates the lastMessage with the latest message received
func (m *LastMessage) ListenUpdates() {
	for msg := range m.Update {
		m.mu.Lock()
		m.Message = msg
		m.mu.Unlock()
	}
}
