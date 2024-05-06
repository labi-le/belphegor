package data

import "sync"

// LastMessage which is stored in Node and serves to identify duplicate messages
type LastMessage struct {
	*Message
	mu     sync.Mutex
	Update chan *Message
}

func NewLastMessage() *LastMessage {
	return &LastMessage{Update: make(chan *Message)}
}

func (m *LastMessage) Get() *Message {
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
