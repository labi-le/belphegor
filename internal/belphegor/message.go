// Package belphegor provides functionality for managing clipboard data between nodes.
package belphegor

import (
	gen "belphegor/internal/belphegor/types"
	"belphegor/pkg/image"
	"belphegor/pkg/pool"
	"bytes"
	"crypto/sha256"
	"errors"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"
	"net/http"
	"runtime"
	"sync"
	"time"
)

type UniqueID = string

var (
	ErrVersionMismatch = errors.New("version mismatch")
)

var (
	currentUniqueID = uuid.New()
	// currentOS represents the operating system information.
	currentOS = &gen.Device{
		Type:   parseDeviceType(runtime.GOOS),
		Arch:   runtime.GOARCH,
		Unique: currentUniqueID.String(),
	}
)

var (
	messagePool = initMessagePool()
	greetPool   = initGreetPool()
)

func initGreetPool() *pool.ObjectPool[*gen.GreetMessage] {
	p := pool.NewObjectPool[*gen.GreetMessage](10)
	p.New = func() *gen.GreetMessage {
		return &gen.GreetMessage{
			UniqueID: currentUniqueID.String(),
			Version:  Version,
			Device:   currentOS,
		}
	}

	return p
}

func initMessagePool() *pool.ObjectPool[*gen.Message] {
	p := pool.NewObjectPool[*gen.Message](10)
	p.New = func() *gen.Message {
		return &gen.Message{
			Header: &gen.Header{
				ID:      uuid.New().String(),
				Device:  currentOS,
				Created: timestamppb.New(time.Now()),
			},
			Data: &gen.Data{},
		}
	}
	return p
}

// AcquireMessage creates a new Message with the provided data.
func AcquireMessage(data []byte) *gen.Message {
	msg := messagePool.Acquire()
	msg.Data.Raw = data
	msg.Data.Hash = hashBytes(data)

	msg.Header.MimeType = parseMimeType(http.DetectContentType(data))

	return msg
}

// ReleaseMessage returns the Message to the messagePool for reuse.
func ReleaseMessage(m *gen.Message) {
	messagePool.Release(m)
}

func parseDeviceType(os string) gen.Type {
	switch os {
	case "linux":
		return gen.Type_Linux
	case "windows":
		return gen.Type_Windows
	case "darwin":
		return gen.Type_MacOS
	case "ios":
		return gen.Type_IOS
	case "android":
		return gen.Type_Android
	default:
		panic("unimplemented device")
	}
}

func HandShake(self *gen.GreetMessage, other *gen.GreetMessage) error {
	if self.Version != other.Version {
		return ErrVersionMismatch
	}

	return nil
}

// lastMessage which is stored in Node and serves to identify duplicate messages
type lastMessage struct {
	*gen.Message
	mu sync.Mutex
}

func (m *lastMessage) Get() *gen.Message {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Message
}

func (m *lastMessage) Set(msg *gen.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Message = msg
}

func parseMimeType(ct string) gen.Mime {
	switch ct {
	case "image/png":
		return gen.Mime_IMAGE
	default:
		return gen.Mime_TEXT
	}
}

// hashBytes calculates the SHA-256 hash of the provided data and returns it as a byte slice.
func hashBytes(data []byte) []byte {
	sha := sha256.New()
	sha.Write(data)

	return sha.Sum(nil)
}

// shortHash returns the first 4 bytes of the provided hash.
func shortHash(oldHash []byte) []byte {
	return oldHash[:4]
}

func MessageIsDuplicate(self *gen.Message, from *gen.Message) bool {
	if self.Header.ID == from.Header.ID {
		return true
	}

	if self.Header.MimeType == gen.Mime_IMAGE && from.Header.MimeType == gen.Mime_IMAGE {
		if equalSystem(self, from) {
			return bytes.Equal(self.Data.Hash, from.Data.Hash)
		}

		// mse: compare images
		identical, err := image.Equal(
			bytes.NewReader(self.Data.Raw),
			bytes.NewReader(from.Data.Raw),
		)
		if err != nil {
			log.Error().Err(err).Msg("failed to compare images")
		}

		return identical
	}

	return false
}

func equalSystem(self *gen.Message, from *gen.Message) bool {
	return self.Header.Device.Type == from.Header.Device.Type &&
		self.Header.Device.Arch == from.Header.Device.Arch
}
