package node

import (
	"sync/atomic"

	"github.com/labi-le/belphegor/internal/types/domain"
)

// Channel is an interface for managing clipboard data
type Channel struct {
	new chan domain.EventMessage
	old atomic.Pointer[domain.EventMessage]
}

func (c *Channel) Send(msg domain.EventMessage) {
	msgRef := &msg

	for {
		old := c.old.Load()
		if old != nil && old.Payload.Duplicate(msgRef.Payload) {
			return
		}

		if c.old.CompareAndSwap(old, msgRef) {
			c.new <- msg
			return
		}
	}
}

func (c *Channel) Listen() <-chan domain.EventMessage {
	return c.new
}

func (c *Channel) Close() error {
	close(c.new)
	return nil
}

func NewChannel() *Channel {
	return &Channel{
		new: make(chan domain.EventMessage),
	}
}
