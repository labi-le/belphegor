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
	"google.golang.org/protobuf/proto"
	"io"
	"net/http"
)

var (
	messagePool     = initMessagePool()
	currentProvider = parseClipboardProvider(clipboard.New())
)

func initMessagePool() *pool.ObjectPool[*Message] {
	p := pool.NewObjectPool[*Message](10)
	p.New = func() *Message {
		return MessageFromProto(&types.Message{
			Header: &types.Header{
				ID:                uuid.New().String(),
				ClipboardProvider: currentProvider,
			},
			Data: &types.Data{},
		})
	}
	return p
}

type Message struct {
	proto *types.Message

	cachedFrom UniqueID
	cachedID   uuid.UUID
}

// MessageFrom creates a new Message with the provided data.
func MessageFrom(data []byte, metadata *MetaData) *Message {
	msg := messagePool.Acquire()
	msg.proto.Data.Raw = data
	msg.proto.Data.Hash = hashBytes(data)

	msg.proto.Header.MimeType = parseMimeType(http.DetectContentType(data))
	msg.proto.Header.From = metadata.UniqueID().String()

	return msg
}

func MessageFromProto(m *types.Message) *Message {
	return &Message{proto: m}
}

func (m *Message) Release() {
	messagePool.Release(m)
}

func (m *Message) Duplicate(new *Message) bool {
	if new == nil || m == nil {
		return false
	}

	if m.proto.Header.MimeType == types.Mime_IMAGE && new.proto.Header.MimeType == types.Mime_IMAGE {
		if m.proto.Header.ClipboardProvider == new.proto.Header.ClipboardProvider {
			return bytes.Equal(m.proto.Data.Hash, new.proto.Data.Hash)
		}

		// mse: compare images
		identical, err := image.EqualMSE(
			bytes.NewReader(m.proto.Data.Raw),
			bytes.NewReader(new.proto.Data.Raw),
		)
		if err != nil {
			log.Error().AnErr("image.EqualMSE", err).Msg("failed to compare images")
		}

		return identical
	}

	return m.proto.Header.ID == new.proto.Header.ID || bytes.Equal(m.proto.Data.Hash, new.proto.Data.Hash)
}

// From who is the owner of this message
func (m *Message) From() UniqueID {
	if m.cachedFrom == uuid.Nil {
		m.cachedFrom = uuid.MustParse(m.proto.Header.From)
	}

	return m.cachedFrom
}

// ID unique message id
func (m *Message) ID() UniqueID {
	if m.cachedID == uuid.Nil {
		m.cachedID = uuid.MustParse(m.proto.Header.ID)
	}

	return m.cachedID
}

func (m *Message) RawData() []byte {
	return m.proto.Data.Raw
}

func (m *Message) Kind() proto.Message {
	return m.proto
}

func (m *Message) Write(writer io.Writer) (int, error) {
	return EncodeWriter(m.Kind(), writer)
}

func ReceiveMessage(conn io.Reader) (*Message, error) {
	message := messagePool.Acquire()
	if err := DecodeReader(conn, message.proto); err != nil {
		return &Message{}, err
	}

	return MessageFromProto(message.proto), nil
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
