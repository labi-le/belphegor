package data

import (
	"bytes"
	"crypto/sha256"
	"github.com/google/uuid"
	"github.com/labi-le/belphegor/internal/types"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/image"
	"github.com/labi-le/belphegor/pkg/pool"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"
	"net/http"
	"time"
)

var (
	messagePool = initMessagePool()
)

func initMessagePool() *pool.ObjectPool[*Message] {
	p := pool.NewObjectPool[*Message](10)
	p.New = func() *Message {
		return MessageFromProto(&types.Message{
			Header: &types.Header{
				ID:                uuid.New().String(),
				Created:           timestamppb.New(time.Now()),
				ClipboardProvider: parseClipboardProvider(clipboard.New()),
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
func MessageFrom(data []byte, metadata *types.Device) *Message {
	msg := messagePool.Acquire()
	msg.Data.Raw = data
	msg.Data.Hash = hashBytes(data)

	msg.Header.MimeType = parseMimeType(http.DetectContentType(data))
	msg.Header.From = metadata.UniqueID

	return msg
}

func MessageFromProto(m *types.Message) *Message {
	return &Message{Message: m}
}

func (m *Message) Release() {
	messagePool.Release(m)
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
