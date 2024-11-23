package domain

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"github.com/labi-le/belphegor/internal/types/proto"
	"github.com/labi-le/belphegor/pkg/image"
	"github.com/labi-le/belphegor/pkg/protoutil"
	"github.com/rs/zerolog/log"
	pb "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io"
	"net"
)

type Message struct {
	Data   Data
	Header Header
}

// MessageFrom creates a new Message with the provided data.
func MessageFrom(data []byte, from UniqueID) Message {
	return Message{
		Data: NewData(data),
		Header: NewHeader(
			WithMime(MimeFromData(data)),
			WithFrom(from),
		),
	}
}

func MessageFromProto(m *proto.Message) Message {
	return Message{
		Data: Data{
			Raw:  m.Data.Raw,
			Hash: m.Data.Hash,
		},
		Header: Header{
			ID:                m.Header.ID,
			Created:           m.Header.Created.AsTime(),
			From:              m.Header.From,
			MimeType:          MimeType(m.Header.MimeType),
			ClipboardProvider: ClipboardProvider(m.Header.ClipboardProvider.Number()),
		},
	}
}

func (m Message) Duplicate(new Message) bool {
	if m.Header.MimeType == MimeTypeImage && new.Header.MimeType == m.Header.MimeType {
		if m.Header.ClipboardProvider == new.Header.ClipboardProvider {
			return bytes.Equal(m.Data.Hash, new.Data.Hash)
		}

		// mse: compare images
		identical, err := image.EqualMSE(
			bytes.NewReader(m.Data.Raw),
			bytes.NewReader(new.Data.Raw),
		)
		if err != nil {
			log.Error().AnErr("image.EqualMSE", err).Msg("failed to compare images")
		}

		return identical
	}

	return m.Header.ID == new.Header.ID || bytes.Equal(m.Data.Hash, new.Data.Hash)
}

func (m Message) From() UniqueID {
	return m.Header.From
}

func (m Message) ID() UniqueID {
	return m.Header.ID
}

func (m Message) RawData() []byte {
	return m.Data.Raw
}

func (m Message) Proto() pb.Message {
	return &proto.Message{
		Data: &proto.Data{
			Raw:  m.Data.Raw,
			Hash: m.Data.Hash,
		},
		Header: &proto.Header{
			From:              m.From(),
			MimeType:          proto.Mime(m.Header.MimeType),
			ID:                m.ID(),
			Created:           timestamppb.New(m.Header.Created),
			ClipboardProvider: proto.Clipboard(m.Header.ClipboardProvider),
		},
	}
}

func (m Message) WriteEncrypted(signer crypto.Signer, writer io.Writer) (int, error) {
	dat, _ := pb.Marshal(m.Proto())
	encrypted, err := signer.Sign(rand.Reader, dat, nil)
	if err != nil {
		return 0, err
	}

	msg := EncryptedMessage{encrypted}
	return protoutil.EncodeWriter(msg, writer)
}

func ReceiveMessage(conn net.Conn, decrypter crypto.Decrypter) (Message, error) {
	var message proto.Message

	var encrypt proto.EncryptedMessage
	if decodeEnc := protoutil.DecodeReader(conn, &encrypt); decodeEnc != nil {
		return Message{}, decodeEnc
	}

	decrypt, decErr := decrypter.Decrypt(rand.Reader, encrypt.Message, nil)
	if decErr != nil {
		return Message{}, decErr
	}

	if err := pb.Unmarshal(decrypt, &message); err != nil {
		return Message{}, err
	}

	return MessageFromProto(&message), nil
}
