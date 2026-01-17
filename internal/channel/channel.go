package channel

import (
	"sync"

	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/id"
)

const historySize = 5

type Channel struct {
	msgMu   sync.RWMutex
	msg     chan domain.EventMessage
	lastMsg domain.EventMessage

	ann         chan domain.EventAnnounce
	fileHistory *history
}

func New(peerMaxCount int) *Channel {
	return &Channel{
		msg:         make(chan domain.EventMessage),
		ann:         make(chan domain.EventAnnounce, peerMaxCount),
		fileHistory: newHistory(historySize),
	}
}

func (c *Channel) LastMsg() domain.EventMessage {
	c.msgMu.RLock()
	defer c.msgMu.RUnlock()
	return c.lastMsg
}

func (c *Channel) Get(msgID id.Unique) (domain.EventMessage, bool) {
	c.msgMu.RLock()
	defer c.msgMu.RUnlock()

	if c.lastMsg.Payload.ID == msgID {
		return c.lastMsg, true
	}

	return domain.EventMessage{}, false
}

func (c *Channel) Send(msg domain.EventMessage) {
	if c.updateLastMsg(msg) {
		c.msg <- msg
	}
}

func (c *Channel) updateLastMsg(msg domain.EventMessage) bool {
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
	if c.fileHistory.Add(ann.Payload.ContentHash, ann) {
		return true
	}

	return false
}

func (c *Channel) Announcements() <-chan domain.EventAnnounce {
	return c.ann
}
