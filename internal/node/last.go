package node

import (
	"github.com/labi-le/belphegor/internal/types/domain"
	"sync"
)

// LastMessage which is stored in Node and serves to identify duplicate messages
type LastMessage struct {
	msg domain.Message
	mu  sync.Mutex
}

func (m *LastMessage) Msg() domain.Message {
	return m.msg
}

func NewLastMessage() *LastMessage {
	return &LastMessage{
		msg: domain.Message{
			Data:   domain.Data{},
			Header: domain.Header{},
		},
	}
}

func (m *LastMessage) Update(msg domain.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msg = msg
}
