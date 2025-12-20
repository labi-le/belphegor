package domain

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"io"
	"net"
	"time"

	"github.com/labi-le/belphegor/internal/types/proto"
	"github.com/labi-le/belphegor/pkg/image"
	"github.com/labi-le/belphegor/pkg/protoutil"
	"github.com/rs/zerolog/log"
	pb "google.golang.org/protobuf/proto"
)

type EventMessage = Event[Message]

type Message struct {
	Data Data
	Mime MimeType
	ID   UniqueID
}

type Data []byte

// MessageFrom creates a new Message with the provided data.
func MessageFrom(data []byte, owner UniqueID) EventMessage {
	return EventMessage{
		Type:    TypeUpdate,
		From:    owner,
		Created: time.Now(),
		Payload: Message{
			Data: data,
			Mime: mimeFromData(data),
			ID:   NewID(),
		},
	}
}

func MessageFromProto(m *proto.Message) EventMessage {
	return NewEvent[Message](
		Message{
			Data: m.Data,
			Mime: MimeType(m.MimeType),
			ID:   m.ID,
		})
}

func (m Message) Duplicate(new Message) bool {
	if m.ID == new.ID {
		return true
	}

	if m.Mime != new.Mime {
		return false
	}

	if m.Mime == MimeTypeImage {
		//if m.Event.ClipboardProvider == new.Event.ClipboardProvider {
		//	return bytes.Equal(m.Data.Raw, new.Data.Raw)
		//}

		identical, err := image.EqualMSE(
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

func (m Message) Proto() pb.Message {
	return &proto.Message{
		ID:       m.ID,
		Data:     m.Data,
		MimeType: proto.Mime(m.Mime),
	}
}

func (m Message) WriteEncrypted(signer crypto.Signer, writer io.Writer) (int, error) {
	dat, _ := pb.Marshal(m.Proto())
	encrypted, err := signer.Sign(rand.Reader, dat, nil)
	if err != nil {
		return 0, err
	}

	msg := EncryptedMessage{encrypted}
	return protoutil.EncodeWriter(msg.Proto(), writer)
}

func ReceiveMessage(conn net.Conn, decrypter crypto.Decrypter) (EventMessage, error) {
	var message proto.Message

	var encrypt proto.EncryptedMessage
	if decodeEnc := protoutil.DecodeReader(conn, &encrypt); decodeEnc != nil {
		return EventMessage{}, decodeEnc
	}

	decrypt, decErr := decrypter.Decrypt(rand.Reader, encrypt.Message, nil)
	if decErr != nil {
		return EventMessage{}, decErr
	}

	if err := pb.Unmarshal(decrypt, &message); err != nil {
		return EventMessage{}, err
	}

	return MessageFromProto(&message), nil
}
