package domain

import (
	"bytes"
	"fmt"
	"time"

	"github.com/labi-le/belphegor/internal/types/proto"
	"github.com/labi-le/belphegor/pkg/id"
	"github.com/labi-le/belphegor/pkg/mime"
	"github.com/labi-le/belphegor/pkg/protoutil"
	"github.com/rs/zerolog/log"
	pb "google.golang.org/protobuf/proto"
)

var (
	_ protoutil.Proto[*proto.Message]          = Message{}
	_ protoutil.Proto[*proto.EncryptedMessage] = EncryptedMessage{}
)

type EventMessage = Event[Message]
type EventEncryptedMessage = Event[EncryptedMessage]

type Data []byte

type Message struct {
	ID   id.Unique
	Data Data
	Mime MimeType
}

func (m Message) Event(owner id.Unique) EventMessage {
	return EventMessage{
		From:    owner,
		Created: time.Now(),
		Payload: m,
	}
}

func NewMessage(data Data) Message {
	return Message{
		ID:   id.New(),
		Data: data,
		Mime: mimeFromData(data),
	}
}

type EncryptedMessage struct {
	ID      id.Unique
	Content []byte
}

func (e EncryptedMessage) Proto() *proto.EncryptedMessage {
	return &proto.EncryptedMessage{
		ID:      e.ID,
		Content: e.Content,
	}
}

// MessageNew creates a new Message with the provided data.
func MessageNew(data []byte, owner id.Unique) EventMessage {
	return EventMessage{
		From:    owner,
		Created: time.Now(),
		Payload: Message{
			Data: data,
			Mime: mimeFromData(data),
			ID:   id.New(),
		},
	}
}

func (m Message) Duplicate(new Message) bool {
	if m.ID == new.ID {
		return true
	}

	if m.Mime != new.Mime {
		return false
	}

	if m.Mime == MimeTypeImage {
		identical, err := mime.EqualMSE(
			bytes.NewReader(m.Data),
			bytes.NewReader(new.Data),
		)
		if err != nil {
			log.Error().AnErr("image.EqualMSE", err).Msg("failed to compare images")
		}

		return identical
	}

	return bytes.Equal(m.Data, new.Data)
}

func (m Message) Proto() *proto.Message {
	return &proto.Message{
		Data:     m.Data,
		MimeType: proto.Mime(m.Mime),
	}
}

type DecryptFn func(encrypted []byte) ([]byte, error)

func MessageFromEncrypted(ev *proto.Event, data Device, fn DecryptFn) (EventMessage, error) {
	payload, ok := ev.Payload.(*proto.Event_Message)
	if ok == false {
		return EventMessage{}, fmt.Errorf("expected: %T, actual: %T", proto.Event_Message{}, ev.Payload)
	}

	decrypted, err := fn(payload.Message.Content)
	if err != nil {
		return EventMessage{}, err
	}

	var msg proto.Message
	if err := pb.Unmarshal(decrypted, &msg); err != nil {
		return EventMessage{}, fmt.Errorf("MessageFromEncrypted: %w", err)
	}

	return EventMessage{
		From:    data.ID,
		Created: ev.Created.AsTime(),
		Payload: Message{
			ID:   payload.Message.ID,
			Data: msg.Data,
			Mime: MimeType(msg.MimeType),
		},
	}, nil
}
