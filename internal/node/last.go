package node

import (
	"github.com/labi-le/belphegor/internal/types/domain"
)

// LastMessage which is stored in Node and serves to identify duplicate messages
type LastMessage struct {
	*domain.Message
	Update chan *domain.Message
}

func NewLastMessage() *LastMessage {
	return &LastMessage{Update: make(chan *domain.Message)}
}

// ListenUpdates updates the lastMessage with the latest message received
func (m *LastMessage) ListenUpdates() {
	for msg := range m.Update {
		m.Message = msg
	}
}
