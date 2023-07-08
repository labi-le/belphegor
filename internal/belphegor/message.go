package belphegor

import (
	"bytes"
	"crypto/sha256"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/vmihailenco/msgpack/v5"
	"io"
	"sync"
)

var messagePool = sync.Pool{
	New: func() interface{} {
		return &Message{
			Header: Header{
				ID: uuid.New(),
			},
			Data: []byte{},
		}
	},
}

type Message struct {
	Header   Header
	Data     []byte
	DataHash []byte
}

type Header struct {
	ID uuid.UUID
}

func (m *Message) Write(w io.Writer) (int, error) {
	return w.Write(encode(m))
}

func NewMessage(data []byte) *Message {
	//return Message{Data: data, Header: Header{
	//	ID: uuid.New(),
	//}}
	msg := messagePool.Get().(*Message)
	msg.Data = data
	msg.DataHash = sha256Hash(data)

	return msg
}

func sha256Hash(data []byte) []byte {
	return sha256.New().Sum(data)
}

func (m *Message) IsDuplicate(msg *Message) bool {
	if msg == nil {
		return false
	}

	return m.Header.ID == msg.Header.ID || bytes.Equal(m.Data, msg.Data) || bytes.Equal(m.DataHash, msg.DataHash)
}

func encode(src interface{}) []byte {
	encoded, err := msgpack.Marshal(src)
	if err != nil {
		log.Error().Msgf("failed to encode clipboard data: %s", err)
	}

	return encoded
}

func decode(r io.Reader, dst interface{}) error {
	return msgpack.NewDecoder(r).Decode(dst)
}
