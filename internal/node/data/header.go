package data

import (
	"crypto/sha256"
	"fmt"
	"github.com/google/uuid"
	"github.com/labi-le/belphegor/internal/types"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

var (
	currentProvider = parseClipboardProvider(clipboard.New())
)

type Header struct {
	ID                uuid.UUID
	Created           time.Time
	From              UniqueID
	MimeType          MimeType
	ClipboardProvider ClipboardProvider
}

func (h *Header) ToProto() *types.Header {
	return &types.Header{
		ID:                h.ID.String(),
		Created:           timestamppb.New(h.Created),
		From:              h.From.String(),
		MimeType:          types.Mime(h.MimeType),
		ClipboardProvider: types.Clipboard(h.ClipboardProvider),
	}
}

func HeaderFromProto(p *types.Header) (Header, error) {
	if p == nil {
		return Header{}, fmt.Errorf("proto header is nil")
	}

	return Header{
		ID:                uuid.MustParse(p.ID),
		Created:           p.Created.AsTime(),
		From:              uuid.MustParse(p.From),
		MimeType:          MimeType(p.MimeType),
		ClipboardProvider: ClipboardProvider(p.ClipboardProvider),
	}, nil
}

var mimeTypeMap = map[string]MimeType{
	"image/png": MimeTypeImage,
	// Добавить другие MIME типы если нужно
}

func parseMimeType(ct string) MimeType {
	if mime, ok := mimeTypeMap[ct]; ok {
		return mime
	}
	return MimeTypeText
}

// hashBytes calculates the SHA-256 hash of the provided data and returns it as a byte slice.
func hashBytes(data []byte) []byte {
	sha := sha256.New()
	sha.Write(data)

	return sha.Sum(nil)
}

type MimeType int32

const (
	MimeTypeText MimeType = iota
	MimeTypeImage
)

type ClipboardProvider int32

const (
	ClipboardNull ClipboardProvider = iota
	ClipboardXSel
	ClipboardXClip
	ClipboardWlClipboard
	ClipboardMasOsStd
	ClipboardWindowsNT10
)

func parseClipboardProvider(m clipboard.Manager) ClipboardProvider {
	switch m.Name() {
	case clipboard.XSel:
		return ClipboardXSel
	case clipboard.XClip:
		return ClipboardXClip
	case clipboard.WlClipboard:
		return ClipboardWlClipboard
	case clipboard.MasOsStd:
		return ClipboardMasOsStd
	case clipboard.WindowsNT10:
		return ClipboardWindowsNT10
	case clipboard.NullClipboard:
		return ClipboardNull
	default:
		log.Fatal().Msgf("unimplemented device: %s", m.Name())
	}

	// unreachable
	return ClipboardNull
}
