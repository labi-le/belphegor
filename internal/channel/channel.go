package channel

import (
	"sync"

	"github.com/labi-le/belphegor/internal/types/domain"
)

type Channel struct {
	msgMu   sync.Mutex
	msg     chan domain.EventMessage
	lastMsg domain.EventMessage

	annMu   sync.Mutex
	ann     chan domain.EventAnnounce
	lastAnn domain.EventAnnounce
}

func New(peerMaxCount int) *Channel {
	return &Channel{
		msg: make(chan domain.EventMessage),
		ann: make(chan domain.EventAnnounce, peerMaxCount),
	}
}

func (c *Channel) LastMsg() domain.EventMessage {
	c.msgMu.Lock()
	defer c.msgMu.Unlock()
	return c.lastMsg
}

func (c *Channel) Send(msg domain.EventMessage) {
	if c.shouldUpdateMsg(msg) {
		c.msg <- msg
	}
}

func (c *Channel) shouldUpdateMsg(msg domain.EventMessage) bool {
	c.msgMu.Lock()
	defer c.msgMu.Unlock()

	if !c.lastMsg.Payload.Zero() && c.lastMsg.Payload.Duplicate(msg.Payload) {
		return false
	}

	c.lastMsg = msg
	return true
}

func (c *Channel) Messages() <-chan domain.EventMessage {
	return c.msg
}

func (c *Channel) Announce(ann domain.EventAnnounce) {
	if c.shouldUpdateAnn(ann) {
		c.ann <- ann
	}
}

func (c *Channel) shouldUpdateAnn(ann domain.EventAnnounce) bool {
	c.annMu.Lock()
	defer c.annMu.Unlock()

	if !c.lastAnn.Payload.Zero() && c.lastAnn.Payload.Duplicate(ann.Payload) {
		return false
	}

	c.lastAnn = ann
	return true
}

func (c *Channel) Announcements() <-chan domain.EventAnnounce {
	return c.ann
}
