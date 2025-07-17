package node

import "github.com/labi-le/belphegor/internal/types/domain"

// Channel is an interface for managing clipboard data
type Channel struct {
	new chan domain.Message
	old domain.Message
}

func (c *Channel) Update(msg domain.Message) {
	c.old = msg
}

func NewChannel() *Channel {
	return &Channel{
		new: make(chan domain.Message),
		old: domain.Message{},
	}
}
