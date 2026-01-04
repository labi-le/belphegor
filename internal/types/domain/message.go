package domain

import (
	"encoding/binary"
	"time"

	"github.com/cespare/xxhash"
	"github.com/labi-le/belphegor/internal/types/proto"
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/id"
	"github.com/labi-le/belphegor/pkg/mime"
	"github.com/labi-le/belphegor/pkg/protoutil"
	"github.com/rs/zerolog"
)

var (
	_ protoutil.Proto[*proto.Message] = Message{}
)

type EventMessage = Event[Message]

type Data []byte

type Message struct {
	ID          id.Unique
	Data        Data
	Mime        mime.Type
	ContentHash uint64
}

func (m Message) Zero() bool {
	return m.ID == 0 || m.ContentHash == 0 || len(m.Data) == 0
}

func (m Message) Event() EventMessage {
	return EventMessage{
		From:    id.MyID,
		Created: time.Now(),
		Payload: m,
	}
}

// MessageNew creates a new Message with the provided data.
func MessageNew(data []byte) EventMessage {
	mt := mime.From(data)
	return EventMessage{
		From:    id.MyID,
		Created: time.Now(),
		Payload: Message{
			Data:        data,
			Mime:        mt,
			ID:          id.New(),
			ContentHash: hashMessage(mt, data),
		},
	}
}

func hashMessage(mt mime.Type, data []byte) uint64 {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(mt))

	d := xxhash.New()
	_, _ = d.Write(buf[:])
	_, _ = d.Write(data)
	return d.Sum64()
}

func (m Message) Duplicate(msg Message) bool {
	if m.ID == msg.ID {
		return true
	}

	if m.Mime != msg.Mime {
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

func (m Message) Proto() *proto.Message {
	return &proto.Message{
		MimeType:      proto.Mime(m.Mime),
		ID:            id.New(),
		ContentLength: int64(len(m.Data)),
		ContentHash:   m.ContentHash,
	}
}

func FromUpdate(update eventful.Update) Message {
	return Message{
		ID:          id.New(),
		Data:        update.Data,
		Mime:        update.MimeType,
		ContentHash: hashMessage(update.MimeType, update.Data),
	}
}

func FromProto(proto *proto.Event, payload *proto.Event_Message, src []byte) EventMessage {
	return EventMessage{
		From:    id.Author(payload.Message.GetID()),
		Created: proto.GetCreated().AsTime(),
		Payload: Message{
			ID:          payload.Message.GetID(),
			Data:        src,
			Mime:        mime.Type(payload.Message.GetMimeType()),
			ContentHash: payload.Message.GetContentHash(),
		},
	}
}

func MsgLogger(base zerolog.Logger, msg Message) zerolog.Logger {
	return base.With().
		Int64("msg_id", msg.ID).
		Int64("node_id", id.Author(msg.ID)).
		Logger()
}
