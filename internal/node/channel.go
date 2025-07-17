package node

import (
	"github.com/labi-le/belphegor/internal/types/domain"
	"sync"
)

// Channel is an interface for managing clipboard data
type Channel struct {
	new chan domain.Message
	old domain.Message
	mu  sync.Mutex
}

func (c *Channel) Send(msg domain.Message) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.old.Duplicate(msg) {
		return
	}
	c.old = msg
	c.new <- msg
}

func (c *Channel) Listen() <-chan domain.Message {
	return c.new
}

func (c *Channel) Close() error {
	close(c.new)
	return nil
}

func NewChannel() *Channel {
	return &Channel{
		new: make(chan domain.Message),
		old: domain.Message{},
	}
}
