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
				Content: []byte{},
			}
		},
	}
)

type Message struct {
	Header  Header
	Content []byte
}

type Header struct {
	ID       uuid.UUID
	MimeType string
	Length   int
	Hash     []byte
}

func (m *Message) Write(w io.Writer) (int, error) {
	return w.Write(encode(m))
}

func NewMessage(data []byte) *Message {
	//return Message{Content: data, Header: Header{
	//	ID: uuid.New(),
	//}}
	msg := messagePool.Get().(*Message)
	msg.Content = data
	msg.Header = Header{
		Hash:     sha256Hash(data),
		MimeType: http.DetectContentType(data),
		Length:   len(data),
		ID:       uuid.New(),
	}

	return msg
}

func sha256Hash(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	return h.Sum(nil)
}

func (m *Message) IsDuplicate(msg *Message) bool {
	if msg == nil {
		return false
	}

	log.Trace().Msgf(
		"compare header %s with %s and hash %x with %x",
		m.Header.ID,
		msg.Header.ID,
		m.Header.Hash,
		msg.Header.Hash,
	)
	return m.Header.ID == msg.Header.ID || bytes.Equal(m.Header.Hash, msg.Header.Hash)
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
