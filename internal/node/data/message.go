package data

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"fmt"
	"github.com/google/uuid"
	"github.com/labi-le/belphegor/internal/types"
	"github.com/labi-le/belphegor/pkg/image"
	"github.com/labi-le/belphegor/pkg/pool/byteslice"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"net/http"
	"time"
)

type Message struct {
	Data   Data
	Header Header
}

func NewMessage(rawData []byte, metadata MetaData) Message {
	return Message{
		Data: Data{
			Raw:  rawData,
			hash: hashBytes(rawData),
		},
		Header: Header{
			ID:                uuid.New(),
			Created:           time.Now(),
			From:              metadata.UniqueID(),
			MimeType:          parseMimeType(http.DetectContentType(rawData)),
			ClipboardProvider: currentProvider,
		},
	}
}

func (m Message) ID() UniqueID {
	return m.Header.ID
}

func (m Message) RawData() []byte {
	return m.Data.Raw
}

func (m Message) ToProto() *types.Message {
	return &types.Message{
		Data:   m.Data.ToProto(),
		Header: m.Header.ToProto(),
	}
}

func MessageFromProto(p *types.Message) (Message, error) {
	if p == nil {
		return Message{}, fmt.Errorf("proto message is nil")
	}

	header, err := HeaderFromProto(p.Header)
	if err != nil {
		return Message{}, fmt.Errorf("failed to convert header: %w", err)
	}

	data, err := DataFromProto(p.Data)
	if err != nil {
		return Message{}, fmt.Errorf("failed to convert data: %w", err)
	}

	return Message{
		Data:   data,
		Header: header,
	}, nil
}

func (m Message) Duplicate(new Message) bool {
	if m.Header.ID == new.Header.ID {
		return true
	}

	if m.Header.MimeType != MimeTypeImage || new.Header.MimeType != MimeTypeImage {
		return m.Data.Equal(new.Data)
	}

	if m.Header.ClipboardProvider == new.Header.ClipboardProvider {
		return m.Data.Equal(new.Data)
	}

	identical, err := image.EqualMSE(
		bytes.NewReader(m.Data.Raw),
		bytes.NewReader(new.Data.Raw),
	)
	if err != nil {
		log.Error().AnErr("image.EqualMSE", err).Msg("failed to compare images")
		return false
	}
	return identical
}

func (m Message) WriteEncrypted(signer crypto.Signer, writer io.Writer) (int, error) {
	protoMsg := m.ToProto()
	size := proto.Size(protoMsg)
	data := byteslice.Get(size)
	defer byteslice.Put(data)

	n, err := proto.MarshalOptions{}.MarshalAppend(data[:0], protoMsg)
	if err != nil {
		return 0, err
	}
	data = data[:len(n)]

	encrypted, err := signer.Sign(rand.Reader, data, nil)
	if err != nil {
		return 0, err
	}

	return EncodeWriter(&types.EncryptedMessage{Message: encrypted}, writer)
}

func ReceiveMessage(conn net.Conn, decrypter crypto.Decrypter) (Message, error) {
	var encrypt types.EncryptedMessage
	if err := DecodeReader(conn, &encrypt); err != nil {
		return Message{}, fmt.Errorf("decode encrypted message: %w", err)
	}

	decrypt, err := decrypter.Decrypt(rand.Reader, encrypt.Message, nil)
	if err != nil {
		return Message{}, fmt.Errorf("decrypt message: %w", err)
	}
	var message types.Message

	if err := proto.Unmarshal(decrypt, &message); err != nil {
		return Message{}, fmt.Errorf("unmarshal message: %w", err)
	}

	return MessageFromProto(&message)
}
