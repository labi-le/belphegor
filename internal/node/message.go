package node

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"github.com/google/uuid"
	"github.com/labi-le/belphegor/internal"
	"github.com/labi-le/belphegor/internal/types"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/image"
	"github.com/labi-le/belphegor/pkg/pool"
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
	// thisDevice represents the current device.
	thisDevice = &types.Device{
		Arch:              runtime.GOARCH,
		UniqueName:        currentUniqueID.String(),
		ClipboardProvider: parseClipboardProvider(clipboard.New()),
	}
)

var (
	messagePool = initMessagePool()
	greetPool   = initGreetPool()
)

func initGreetPool() *pool.ObjectPool[*types.GreetMessage] {
	p := pool.NewObjectPool[*types.GreetMessage](10)
	p.New = func() *types.GreetMessage {
		return &types.GreetMessage{
			UniqueID: currentUniqueID.String(),
			Version:  internal.Version,
			Device:   thisDevice,
		}
	}

	return p
}

func initMessagePool() *pool.ObjectPool[*types.Message] {
	p := pool.NewObjectPool[*types.Message](10)
	p.New = func() *types.Message {
		return &types.Message{
			Header: &types.Header{
				ID:      uuid.New().String(),
				Device:  thisDevice,
				Created: timestamppb.New(time.Now()),
			},
			Data: &types.Data{},
		}
	}
	return p
}

// MessageFrom creates a new Message with the provided data.
func MessageFrom(data []byte) *types.Message {
	msg := messagePool.Acquire()
	msg.Data.Raw = data
	msg.Data.Hash = hashBytes(data)

	msg.Header.MimeType = parseMimeType(http.DetectContentType(data))

	return msg
}

func parseClipboardProvider(m clipboard.Manager) types.Clipboard {
	switch m.Name() {
	case clipboard.XSel:
		return types.Clipboard_XSel
	case clipboard.XClip:
		return types.Clipboard_XClip
	case clipboard.WlClipboard:
		return types.Clipboard_WlClipboard
	case clipboard.MasOsStd:
		return types.Clipboard_MasOsStd
	case clipboard.WindowsNT10:
		return types.Clipboard_WindowsNT10

	default:
		panic("unimplemented device")
	}
}

func HandShake(self *types.GreetMessage, other *types.GreetMessage) error {
	if self.Version != other.Version {
		return ErrVersionMismatch
	}

	return nil
}

// lastMessage which is stored in Node and serves to identify duplicate messages
type lastMessage struct {
	*types.Message
	mu sync.Mutex
}

func (m *lastMessage) Get() *types.Message {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Message
}

func (m *lastMessage) Set(msg *types.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Message = msg
}

func parseMimeType(ct string) types.Mime {
	switch ct {
	case "image/png":
		return types.Mime_IMAGE
	default:
		return types.Mime_TEXT
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

func MessageIsDuplicate(self *types.Message, from *types.Message) bool {
	if self.Header.ID == from.Header.ID {
		return true
	}

	if self.Header.MimeType == types.Mime_IMAGE && from.Header.MimeType == types.Mime_IMAGE {
		if equalClipboardProviders(self, from) {
			return bytes.Equal(self.Data.Hash, from.Data.Hash)
		}

		// mse: compare images
		identical, err := image.EqualMSE(
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

func equalClipboardProviders(self *types.Message, from *types.Message) bool {
	return self.Header.Device.ClipboardProvider == from.Header.Device.ClipboardProvider
}
