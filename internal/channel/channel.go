package channel

import (
	"sync"

	"github.com/labi-le/belphegor/internal/types/domain"
)

const HistorySize = 5

type Channel struct {
	msgMu   sync.RWMutex
	msg     chan domain.EventMessage
	lastMsg domain.EventMessage

	ann chan domain.EventAnnounce

	fileHistory *announceHistory
	servedFiles *servedFilesHistory
}

func New(peerMaxCount int) *Channel {
	return &Channel{
		msg:         make(chan domain.EventMessage),
		ann:         make(chan domain.EventAnnounce, peerMaxCount),
		fileHistory: newHistory(HistorySize),
		servedFiles: newServedFilesHistory(HistorySize),
	}
}

func (c *Channel) LastMsg() domain.EventMessage {
	c.msgMu.RLock()
	defer c.msgMu.RUnlock()
	return c.lastMsg
}

func (c *Channel) Get(msgID domain.MessageID) (domain.EventMessage, bool) {
	c.msgMu.RLock()
	defer c.msgMu.RUnlock()

	if c.lastMsg.Payload.ID == msgID {
		msg := c.lastMsg
		return msg, true
	}

	return c.servedFiles.Get(msgID)
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

	if msg.Payload.MimeType.IsPath() {
		c.servedFiles.Add(msg.Payload.ID, msg)
	}
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
	return c.fileHistory.Add(ann.Payload.ContentHash, ann)
}

func (c *Channel) Announcements() <-chan domain.EventAnnounce {
	return c.ann
}
