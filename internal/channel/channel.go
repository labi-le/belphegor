package channel

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
	metadataMsg := msg
	metadataMsg.Payload.Data = nil

	for {
		old := c.old.Load()
		if old != nil && old.Payload.Duplicate(msg.Payload) {
			return
		}

		if c.old.CompareAndSwap(old, &metadataMsg) {
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

func New() *Channel {
	return &Channel{
		new: make(chan domain.EventMessage),
	}
}
