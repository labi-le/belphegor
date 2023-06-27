package belphegor

import (
	"bytes"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/vmihailenco/msgpack/v5"
	"io"
)

type Header struct {
	ID   uuid.UUID
	From string
}

type Message struct {
	Header Header
	Data   []byte
}

func (m Message) Write(w io.Writer) (int, error) {
	return w.Write(encode(m))
}

func NewMessage(data []byte, addr string) Message {
	return Message{Data: data, Header: Header{
		ID:   uuid.New(),
		From: addr,
	}}
}

func (m Message) IsDuplicate(msg Message) bool {
	return m.Header.ID == msg.Header.ID && m.Header.From == msg.Header.From && bytes.Equal(m.Data, msg.Data)
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
