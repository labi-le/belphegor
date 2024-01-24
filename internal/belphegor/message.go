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
	"time"
)

// messagePool is a pool for reusing Message objects.
var messagePool = sync.Pool{
	New: func() interface{} {
		return &Message{
			Header: Header{
				ID:      uuid.New(),
				OS:      currentOS,
				Created: time.Now(),
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
	Created  time.Time
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

// AcquireMessage creates a new Message with the provided data.
func AcquireMessage(data []byte) *Message {
	msg := messagePool.Get().(*Message)
	msg.Data = Data{
		Raw:  data,
		Hash: hash(data),
	}
	msg.Header.MimeType = http.DetectContentType(data)

	return msg
}

// Write writes the encoded Message to an io.Writer.
func (m *Message) Write(w io.Writer) (int, error) {
	return w.Write(encode(m))
}

// Release returns the Message to the messagePool for reuse.
func (m *Message) Release() {
	messagePool.Put(m)
}

func (m *Message) IsImage() bool {
	return image.HasPicture(m.Header.MimeType)
}

// encode encodes the source interface using msgpack and returns the encoded byte slice.
func encode(src interface{}) []byte {
	encoded, err := msgpack.Marshal(src)
	if err != nil {
		log.Error().Msgf("failed to encode clipboard data: %s", err)
	}

	return encoded
}

// decodeMessage decodes data from an io.Reader into the destination interface using msgpack.
func decodeMessage(r io.Reader, msg *Message) error {
	return decode(r, msg)
}

func decode(r io.Reader, dst interface{}) error {
	decoder := msgpack.GetDecoder()
	decoder.Reset(r)
	defer msgpack.PutDecoder(decoder)

	return decoder.Decode(dst)
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

func MessageIsDuplicate(msg1 *Message, msg2 *Message) bool {
	if msg1.Header.ID == msg2.Header.ID {
		return true
	}

	if msg1.IsImage() && msg2.IsImage() {
		if equalSystem(msg1, msg2) {
			return bytes.Equal(msg1.Data.Hash, msg2.Data.Hash)
		}

		// mse: compare images
		identical, err := image.Equal(
			bytes.NewReader(msg1.Data.Raw),
			bytes.NewReader(msg2.Data.Raw),
		)
		if err != nil {
			log.Error().Err(err).Msg("failed to compare images")
		}

		return identical
	}

	return false
}

func equalSystem(msg1 *Message, msg2 *Message) bool {
	return msg1.Header.OS.Name == msg2.Header.OS.Name &&
		msg1.Header.OS.Arch == msg2.Header.OS.Arch
}
