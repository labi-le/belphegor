package belphegor

import (
	"bytes"
	"crypto/sha1"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/vmihailenco/msgpack/v5"
	"io"
	"net/http"
	"runtime"
	"sync"
)

var messagePool = sync.Pool{
	New: func() interface{} {
		return &Message{
			Header: Header{
				ID: uuid.New(),
				OS: os,
			},
			Data: Data{},
		}
	},
}

var os = &OS{
	Name: runtime.GOOS,
	Arch: runtime.GOARCH,
}

type Message struct {
	Data   Data
	Header Header
}

type Data struct {
	Raw  []byte
	Hash []byte
}

type OS struct {
	Name string
	Arch string
}

type Header struct {
	OS       *OS
	MimeType string
	ID       uuid.UUID
}

func (m *Message) Write(w io.Writer) (int, error) {
	return w.Write(encode(m))
}

func NewMessage(data []byte) *Message {
	msg := messagePool.Get().(*Message)
	msg.Data = Data{
		Raw:  data,
		Hash: hash(data),
	}
	msg.Header.MimeType = http.DetectContentType(data)

	return msg
}

func (m *Message) IsDuplicate(msg Message) bool {
	return m.Header.ID == msg.Header.ID || bytes.Equal(m.Data.Hash, msg.Data.Hash)
}

func (m *Message) Free() {
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

func hash(data []byte) []byte {
	sha := sha1.New() //nolint:gosec
	sha.Write(data)

	return sha.Sum(nil)
}

func shortHash(oldHash []byte) []byte {
	return oldHash[:4]
}
