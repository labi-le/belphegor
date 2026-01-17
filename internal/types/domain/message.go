package domain

import (
	"encoding/binary"
	"hash"
	"sync"
	"time"

	"github.com/cespare/xxhash"
	"github.com/dustin/go-humanize"
	"github.com/labi-le/belphegor/pkg/id"
	"github.com/labi-le/belphegor/pkg/mime"
	"github.com/rs/zerolog"
)

type EventMessage = Event[Message]

type (
	MessageID = id.Unique
	Data      []byte
)

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
		From:    id.MyID,
		Created: time.Now(),
		Payload: m,
	}
}

// MessageNew creates a new Message with the provided data.
// Deprecated: Use MessageFromUpdate instead.
func MessageNew(data []byte) EventMessage {
	mt := mime.From(data)
	return EventMessage{
		From:    id.MyID,
		Created: time.Now(),
		Payload: Message{
			Data:        data,
			MimeType:    mt,
			ID:          id.New(),
			ContentHash: hashMessage(mt, data),
		},
	}
}

var hasherPool = sync.Pool{
	New: func() any {
		return xxhash.New()
	},
}

func hashMessage(mt mime.Type, data []byte) uint64 {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(mt))

	d := hasherPool.Get().(hash.Hash64)
	defer func() {
		d.Reset()
		hasherPool.Put(d)
	}()

	_, _ = d.Write(buf[:])
	_, _ = d.Write(data)
	return d.Sum64()
}

func (m Message) Duplicate(msg Message) bool {
	if m.ID == msg.ID {
		return true
	}

	if m.MimeType != msg.MimeType {
		return false
	}

	return m.ContentHash != 0 && m.ContentHash == msg.ContentHash

	//if m.Mime.IsImage() {
	//	identical, err := mime.EqualMSE(
	//		bytes.NewReader(m.Data),
	//		bytes.NewReader(new.Data),
	//	)
	//	if err != nil {
	//		log.Error().AnErr("image.EqualMSE", err).Msg("failed to compare images")
	//	}
	//
	//	return identical
	//}
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
	e.Int64("id", m.ID)
	e.Int64("node", id.Author(m.ID))
	e.Stringer("mime", m.MimeType)
	e.Str("size", humanize.Bytes(m.ContentLength))
	e.Uint64("hash", m.ContentHash)
}
