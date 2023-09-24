// Package belphegor provides functionality for managing clipboard data between nodes.
package belphegor

import (
	"belphegor/pkg/image"
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

// messagePool is a pool for reusing Message objects.
var messagePool = sync.Pool{
	New: func() interface{} {
		return &Message{
			Header: Header{
				ID: uuid.New(),
				OS: currentOS,
			},
			Data: Data{},
		}
	},
}

// currentOS represents the operating system information.
var currentOS = &OS{
	Name: runtime.GOOS,
	Arch: runtime.GOARCH,
}

// Message represents clipboard data and its associated metadata.
type Message struct {
	Data   Data   // Clipboard data
	Header Header // Metadata
}

// Header represents the metadata associated with a Message.
type Header struct {
	OS       *OS
	MimeType string
	ID       uuid.UUID
}

// Data represents the clipboard data and its SHA-1 hash.
type Data struct {
	Raw  []byte // Raw clipboard data
	Hash []byte // SHA-1 hash of the clipboard data
}

// OS represents the operating system information.
type OS struct {
	Name string // Name of the operating system
	Arch string // Architecture of the operating system
}

// Write writes the encoded Message to an io.Writer.
func (m *Message) Write(w io.Writer) (int, error) {
	return w.Write(encode(m))
}

// NewMessage creates a new Message with the provided data.
func NewMessage(data []byte) *Message {
	msg := messagePool.Get().(*Message)
	msg.Data = Data{
		Raw:  data,
		Hash: hash(data),
	}
	msg.Header.MimeType = http.DetectContentType(data)

	return msg
}

// IsDuplicate checks if the Message is a duplicate of another Message.
func (m *Message) IsDuplicate(msg Message) bool {
	// If the MIME type is image/png, compare the images
	// windows and linux are image processed differently and comparing hashes becomes meaningless
	if m.hasSameMimeType(msg) && m.isPicture(msg) {
		return imageComparison(msg, m)
	}

	return m.Header.ID == msg.Header.ID || bytes.Equal(m.Data.Hash, msg.Data.Hash)
}

func (m *Message) isPicture(msg Message) bool {
	switch msg.Header.MimeType {
	case "image/png":
		return true
	case "image/jpeg":
		return true
	case "image/gif":
		return true
	default:
		return false
	}
}

func (m *Message) hasSameMimeType(msg Message) bool {
	return m.Header.MimeType == msg.Header.MimeType
}

func imageComparison(msg Message, m *Message) bool {
	identical, err := image.IsDuplicate(msg.Data.Raw, m.Data.Raw)
	if err != nil {
		log.Error().Err(err).Msg("failed to compare images")
	}

	return identical
}

// Free returns the Message to the messagePool for reuse.
func (m *Message) Free() {
	messagePool.Put(m)
}

// encode encodes the source interface using msgpack and returns the encoded byte slice.
func encode(src interface{}) []byte {
	encoded, err := msgpack.Marshal(src)
	if err != nil {
		log.Error().Msgf("failed to encode clipboard data: %s", err)
	}

	return encoded
}

// decode decodes data from an io.Reader into the destination interface using msgpack.
func decode(r io.Reader, dst interface{}) error {
	return msgpack.NewDecoder(r).Decode(dst)
}

// hash calculates the SHA-1 hash of the provided data and returns it as a byte slice.
func hash(data []byte) []byte {
	sha := sha1.New() //nolint:gosec
	sha.Write(data)

	return sha.Sum(nil)
}

// shortHash returns the first 4 bytes of the provided hash.
func shortHash(oldHash []byte) []byte {
	return oldHash[:4]
}
