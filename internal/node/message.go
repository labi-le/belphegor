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
		Name:     deviceName(),
		Arch:     runtime.GOARCH,
		UniqueID: uuid.New().String(),
	}

	clipboardManager = parseClipboardProvider(clipboard.New())
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

func initMessagePool() *pool.ObjectPool[*Message] {
	p := pool.NewObjectPool[*Message](10)
	p.New = func() *Message {
		return MessageFromProto(&types.Message{
			Header: &types.Header{
				From:              thisDevice.UniqueID,
				ID:                uuid.New().String(),
				Created:           timestamppb.New(time.Now()),
				ClipboardProvider: clipboardManager,
			},
			Data: &types.Data{},
		})
	}
	return p
}

type Message struct {
	*types.Message
}

// MessageFrom creates a new Message with the provided data.
func MessageFrom(data []byte) *Message {
	msg := messagePool.Acquire()
	msg.Data.Raw = data
	msg.Data.Hash = hashBytes(data)

	msg.Header.MimeType = parseMimeType(http.DetectContentType(data))
	msg.Header.From = thisDevice.UniqueID

	return msg
}

func MessageFromProto(m *types.Message) *Message {
	return &Message{Message: m}
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
	*Message
	mu     sync.Mutex
	update chan *Message
}

func NewLastMessage() *LastMessage {
	return &LastMessage{update: make(chan *Message)}
}

func (m *LastMessage) Get() *Message {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Message
}

// ListenUpdates updates the lastMessage with the latest message received
func (m *LastMessage) ListenUpdates() {
	for msg := range m.update {
		m.mu.Lock()
		m.Message = msg
		m.mu.Unlock()
	}
}

func (m *Message) Duplicate(new *Message) bool {
	if new == nil || m == nil {
		return false
	}

	if m.Header.MimeType == types.Mime_IMAGE && new.Header.MimeType == types.Mime_IMAGE {
		if m.Header.ClipboardProvider == new.Header.ClipboardProvider {
			return bytes.Equal(m.Data.Hash, new.Data.Hash)
		}

		// mse: compare images
		identical, err := image.EqualMSE(
			bytes.NewReader(m.Data.Raw),
			bytes.NewReader(new.Data.Raw),
		)
		if err != nil {
			log.Error().AnErr("image.EqualMSE", err).Msg("failed to compare images")
		}

		return identical
	}

	return m.Header.ID == new.Header.ID || bytes.Equal(m.Data.Hash, new.Data.Hash)
}

func (m *Message) Me() bool {
	return m.Header.From == thisDevice.UniqueID
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
