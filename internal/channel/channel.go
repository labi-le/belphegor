package channel

import (
	"sync/atomic"

	"github.com/labi-le/belphegor/internal/types/domain"
)

type Channel struct {
	msg     chan domain.EventMessage
	lastMsg atomic.Pointer[domain.EventMessage]

	ann     chan domain.EventAnnounce
	lastAnn atomic.Pointer[domain.EventAnnounce]
}

func (c *Channel) LastMsg() domain.EventMessage {
	return *c.lastMsg.Load()
}

func New() *Channel {
	return &Channel{
		msg: make(chan domain.EventMessage),
		ann: make(chan domain.EventAnnounce, 100),
	}
}

func (c *Channel) Send(msg domain.EventMessage) {
	for {
		old := c.lastMsg.Load()
		if old != nil && old.Payload.Duplicate(msg.Payload) {
			return
		}

		if c.lastMsg.CompareAndSwap(old, &msg) {
			c.msg <- msg
			return
		}
	}
}

func (c *Channel) Messages() <-chan domain.EventMessage {
	return c.msg
}

func (c *Channel) Announce(ann domain.EventAnnounce) {
	for {
		old := c.lastAnn.Load()

		if old != nil && old.Payload.ID == ann.Payload.ID && old.From == ann.From {
			return
		}

		if c.lastAnn.CompareAndSwap(old, &ann) {
			select {
			case c.ann <- ann:
			default:
			}
			return
		}
	}
}

func (c *Channel) Announcements() <-chan domain.EventAnnounce {
	return c.ann
}

func (c *Channel) Close() error {
	close(c.msg)
	close(c.ann)
	return nil
}
