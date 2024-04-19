package node

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"github.com/google/uuid"
	"github.com/labi-le/belphegor/internal"
	"github.com/labi-le/belphegor/internal/types"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/image"
	"github.com/labi-le/belphegor/pkg/pool"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"
	"net/http"
	"os"
	"os/user"
	"runtime"
	"sync"
	"time"
)

type UniqueID = string

var (
	// thisDevice represents the current device.
	thisDevice = &types.Device{
		Name:              deviceName(),
		Arch:              runtime.GOARCH,
		UniqueID:          uuid.New().String(),
		ClipboardProvider: parseClipboardProvider(clipboard.New()),
	}
)

func deviceName() string {
	hostname, hostErr := os.Hostname()
	if hostErr != nil {
		log.Error().AnErr("deviceName:hostname", hostErr)
		return "unknown@unknown"
	}

	current, userErr := user.Current()
	if userErr != nil {
		log.Error().AnErr("deviceName:username", userErr)

		return "unknown@unknown"
	}

	return fmt.Sprintf("%s@%s", current.Username, hostname)
}

var (
	messagePool = initMessagePool()
	greetPool   = initGreetPool()
)

func initGreetPool() *pool.ObjectPool[*types.GreetMessage] {
	p := pool.NewObjectPool[*types.GreetMessage](10)
	p.New = func() *types.GreetMessage {
		return &types.GreetMessage{
			Version: internal.Version,
			Device:  thisDevice,
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
		log.Fatal().Msgf("unimplemented device: %s", m.Name())
	}

	// unreachable
	return 0
}

// LastMessage which is stored in Node and serves to identify duplicate messages
type LastMessage struct {
	msg    *types.Message
	mu     sync.Mutex
	update chan *types.Message
}

func NewLastMessage() *LastMessage {
	return &LastMessage{update: make(chan *types.Message)}
}

func (m *LastMessage) Get() *types.Message {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.msg
}

// ListenUpdates updates the lastMessage with the latest message received
func (m *LastMessage) ListenUpdates() {
	for msg := range m.update {
		m.mu.Lock()
		m.msg = msg
		m.mu.Unlock()
	}
}

func (m *LastMessage) Duplicate(new *types.Message, from *types.Device, self *types.Device) bool {
	if new == nil || m.msg == nil {
		return false
	}

	message := m.Get()
	if message.Header.MimeType == types.Mime_IMAGE && new.Header.MimeType == types.Mime_IMAGE {
		if self.ClipboardProvider == from.ClipboardProvider {
			return bytes.Equal(message.Data.Hash, new.Data.Hash)
		}

		// mse: compare images
		identical, err := image.EqualMSE(
			bytes.NewReader(message.Data.Raw),
			bytes.NewReader(new.Data.Raw),
		)
		if err != nil {
			log.Error().AnErr("image.EqualMSE", err).Msg("failed to compare images")
		}

		return identical
	}

	return message.Header.ID == new.Header.ID
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
