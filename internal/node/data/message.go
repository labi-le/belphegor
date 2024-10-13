package data

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"github.com/google/uuid"
	"github.com/labi-le/belphegor/internal/types"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/image"
	"github.com/labi-le/belphegor/pkg/pool"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io"
	"net"
	"net/http"
	"time"
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
				Created:           timestamppb.New(time.Now()),
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

func (m *Message) From() UniqueID {
	if m.cachedFrom == uuid.Nil {
		m.cachedFrom = uuid.MustParse(m.proto.Header.From)
	}

	return m.cachedFrom
}

func (m *Message) ID() UniqueID {
	if m.cachedFrom == uuid.Nil {
		m.cachedFrom = uuid.MustParse(m.proto.Header.From)
	}

	return m.cachedFrom
}

func (m *Message) RawData() []byte {
	return m.proto.Data.Raw
}

func (m *Message) Kind() proto.Message {
	return m.proto
}

func (m *Message) WriteEncrypted(signer crypto.Signer, writer io.Writer) (int, error) {
	dat, _ := proto.Marshal(m.Kind())
	encrypted, err := signer.Sign(rand.Reader, dat, nil)
	if err != nil {
		return 0, err
	}

	return EncodeWriter(&types.EncryptedMessage{Message: encrypted}, writer)
}

func ReceiveMessage(conn net.Conn, decrypter crypto.Decrypter) (*Message, error) {
	var message types.Message

	var encrypt types.EncryptedMessage
	if decodeEnc := DecodeReader(conn, &encrypt); decodeEnc != nil {
		return &Message{}, decodeEnc
	}

	decrypt, decErr := decrypter.Decrypt(rand.Reader, encrypt.Message, nil)
	if decErr != nil {
		return &Message{}, decErr
	}

	if err := proto.Unmarshal(decrypt, &message); err != nil {
		return &Message{}, err
	}

	return MessageFromProto(&message), nil
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
	case clipboard.NullClipboard:
		return types.Clipboard_Null

	default:
		log.Fatal().Msgf("unimplemented device: %s", m.Name())
	}

	// unreachable
	return 0
}
