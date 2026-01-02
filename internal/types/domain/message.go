package domain

import (
	"bytes"
	"time"

	"github.com/labi-le/belphegor/internal/types/proto"
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/id"
	"github.com/labi-le/belphegor/pkg/mime"
	"github.com/labi-le/belphegor/pkg/protoutil"
)

var (
	_ protoutil.Proto[*proto.Message] = Message{}
)

type EventMessage = Event[Message]

type Data []byte

type Message struct {
	ID   id.Unique
	Data Data
	Mime mime.Type
}

func (m Message) Event(owner id.Unique) EventMessage {
	return EventMessage{
		From:    owner,
		Created: time.Now(),
		Payload: m,
	}
}

// MessageNew creates a new Message with the provided data.
func MessageNew(data []byte, owner id.Unique) EventMessage {
	return EventMessage{
		From:    owner,
		Created: time.Now(),
		Payload: Message{
			Data: data,
			Mime: mime.From(data),
			ID:   id.New(),
		},
	}
}

func (m Message) Duplicate(msg Message) bool {
	if m.ID == msg.ID {
		return true
	}

	if m.Mime != msg.Mime {
		return false
	}

	if bytes.Equal(m.Data, msg.Data) {
		return true
	}

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

	return false
}

func (m Message) Proto() *proto.Message {
	return &proto.Message{
		MimeType:      proto.Mime(m.Mime),
		ID:            id.New(),
		ContentLength: int64(len(m.Data)),
	}
}

func FromUpdate(update eventful.Update) Message {
	return Message{
		ID:   id.New(),
		Data: update.Data,
		Mime: update.MimeType,
	}
}

func FromProto(from id.Unique, proto *proto.Event, payload *proto.Event_Message, src []byte) EventMessage {
	return EventMessage{
		From:    from,
		Created: proto.GetCreated().AsTime(),
		Payload: Message{
			ID:   payload.Message.GetID(),
			Data: src,
			Mime: mime.Type(payload.Message.GetMimeType()),
		},
	}
}
