package domain

import (
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/labi-le/belphegor/pkg/id"
	"github.com/labi-le/belphegor/pkg/mime"
	"github.com/rs/zerolog"
)

type EventMessage = Event[Message]

type (
	Data []byte
)

// MessageID strict type definition
type MessageID id.Unique

func NewMessageID() MessageID {
	return MessageID(id.New())
}

func (m MessageID) String() string {
	return strconv.FormatInt(int64(m), 10)
}

func (m MessageID) Int64() int64 {
	return int64(m)
}

func (m MessageID) Zero() bool {
	return m == 0
}

func (m MessageID) MarshalZerologObject(e *zerolog.Event) {
	e.Int64("msg_id", int64(m))
}

type Message struct {
	ID            MessageID
	Data          Data
	MimeType      mime.Type
	ContentHash   uint64
	ContentLength uint64
	Name          string
}

func (m Message) Zero() bool {
	return m.ID == 0 || m.ContentHash == 0 || m.ContentLength == 0
}

func (m Message) Event() EventMessage {
	return EventMessage{
		From:    NodeID(id.MyID),
		Created: time.Now(),
		Payload: m,
	}
}

func (m Message) Duplicate(msg Message) bool {
	if m.ID == msg.ID {
		return true
	}

	if m.MimeType != msg.MimeType {
		return false
	}

	return m.ContentHash != 0 && m.ContentHash == msg.ContentHash
}

func (m Message) DuplicateByAnnounce(ann Announce) bool {
	if m.ID == ann.ID {
		return true
	}

	return m.ContentHash != 0 && m.ContentHash == ann.ContentHash
}

func (m Message) Announce() Announce {
	return Announce{
		ID:            m.ID,
		MimeType:      m.MimeType,
		ContentHash:   m.ContentHash,
		ContentLength: m.ContentLength,
	}
}

func (m Message) MarshalZerologObject(e *zerolog.Event) {
	e.Object("id", m.ID)
	e.Int64("node", id.Author(id.Unique(m.ID)))
	e.Stringer("mime", m.MimeType)
	e.Str("size", humanize.Bytes(m.ContentLength))
	e.Uint64("hash", m.ContentHash)
}
