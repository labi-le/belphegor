package belphegor

import (
	"bytes"
	"crypto/sha256"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/vmihailenco/msgpack/v5"
	"io"
	"net/http"
	"sync"
)

var (
	messagePool = sync.Pool{
		New: func() interface{} {
			return &Message{
				Header: Header{
					ID: uuid.New(),
				},
				Data: Data{},
			}
		},
	}
)

type Data struct {
	Content  []byte
	Hash     []byte
	MimeType string
	Length   int
}

func (d Data) DecodeMsgpack(decoder *msgpack.Decoder) error {
	err := decoder.Decode(&d)
	if err != nil {
		return err
	}

	d.Hash = sha256Hash(d.Content)
	d.MimeType = http.DetectContentType(d.Content)
	d.Length = len(d.Content)

	return nil
}

func NewData(content []byte) Data {
	return Data{
		Content:  content,
		Hash:     sha256Hash(content),
		MimeType: http.DetectContentType(content),
		Length:   len(content),
	}
}

type Message struct {
	Header Header
	Data   Data
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
	msg.Data = NewData(data)

	return msg
}

func sha256Hash(data []byte) []byte {
	return sha256.New().Sum(data)
}

func (m *Message) IsDuplicate(msg *Message) bool {
	if msg == nil {
		return false
	}

	log.Trace().Msgf(
		"compare header %s with %s and hash %s with %s",
		m.Header.ID,
		msg.Header.ID,
		m.Data.Hash,
		msg.Data.Hash,
	)
	return m.Header.ID == msg.Header.ID || bytes.Equal(m.Data.Hash, msg.Data.Hash)
}

func (m *Message) Release() {
	messagePool.Put(m)
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
