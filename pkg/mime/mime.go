package mime

import (
	"bytes"
	"strings"
)

type (
	Type    int32
	TypeMap map[string]Type
)

const (
	TypeUnknown Type = iota - 1

	TypeText
	TypeImage
	TypePath

	TypeAudio
	TypeVideo
	TypeBinary
)

func (t Type) IsImage() bool { return t == TypeImage }
func (t Type) IsText() bool  { return t == TypeText }
func (t Type) IsPath() bool  { return t == TypePath }

func (t Type) String() string {
	switch t {
	case TypeText:
		return "text"
	case TypeImage:
		return "image"
	case TypePath:
		return "path"
	case TypeAudio:
		return "audio"
	case TypeVideo:
		return "video"
	case TypeBinary:
		return "binary"
	default:
		return "unknown"
	}
}

var (
	imageTypes = TypeMap{
		"image/png":  TypeImage,
		"image/jpeg": TypeImage,
		"image/jpg":  TypeImage,
		"image/gif":  TypeImage,
		"image/bmp":  TypeImage,
		"image/webp": TypeImage,
	}

	textTypes = TypeMap{
		"text/plain":               TypeText,
		"text/plain;charset=utf-8": TypeText,
		"utf8_string":              TypeText,
		"text":                     TypeText,
		"string":                   TypeText,
	}

	pathTypes = TypeMap{
		"text/uri-list":                TypePath,
		"application/x-cf-hdrop":       TypePath,
		"application/x-ms-hdrop":       TypePath,
		"x-special/gnome-copied-files": TypePath,
	}

	supportedTypes TypeMap
)

func init() {
	supportedTypes = make(TypeMap, len(imageTypes)+len(textTypes)+len(pathTypes))
	for k := range imageTypes {
		supportedTypes[k] = imageTypes[k]
	}
	for k := range textTypes {
		supportedTypes[k] = textTypes[k]
	}
	for k := range pathTypes {
		supportedTypes[k] = pathTypes[k]
	}
}

func SupportedTypes() TypeMap { return supportedTypes }

func IsSupported(mimeType string) bool {
	_, ok := supportedTypes[strings.ToLower(mimeType)]
	return ok
}

func AsType(mimeType string) Type {
	typ, ok := supportedTypes[strings.ToLower(mimeType)]
	if !ok {
		return TypeUnknown
	}
	return typ
}

func normalizeMime(ct string) string {
	ct = strings.ToLower(strings.TrimSpace(ct))
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	return ct
}

func classifyMime(ct string) Type {
	ct = normalizeMime(ct)

	if v, ok := supportedTypes[ct]; ok {
		return v
	}

	switch {
	case strings.HasPrefix(ct, "image/"):
		return TypeImage
	case strings.HasPrefix(ct, "text/"):
		return TypeText
	case strings.HasPrefix(ct, "video/"):
		return TypeVideo
	case strings.HasPrefix(ct, "audio/"):
		return TypeAudio
	case strings.HasPrefix(ct, "application/"):
		return TypeBinary
	default:
		if ct == "" {
			return TypeUnknown
		}
		return TypeBinary
	}
}

func fromBytesSniff(data []byte) string {
	switch {
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x89, 0x50, 0x4E, 0x47}):
		return "image/png"
	case len(data) >= 2 && bytes.Equal(data[:2], []byte{0xFF, 0xD8}):
		return "image/jpeg"
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x47, 0x49, 0x46, 0x38}):
		return "image/gif"
	case len(data) >= 2 && bytes.Equal(data[:2], []byte{0x42, 0x4D}):
		return "image/bmp"
	case len(data) >= 12 && bytes.Equal(data[8:12], []byte("WEBP")):
		return "image/webp"
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x25, 0x50, 0x44, 0x46}):
		return "application/pdf"
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x50, 0x4B, 0x03, 0x04}):
		return "application/zip"
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x52, 0x61, 0x72, 0x21}):
		return "application/x-rar-compressed"
	case len(data) >= 2 && bytes.Equal(data[:2], []byte{0x1F, 0x8B}):
		return "application/gzip"
	default:
		return "text"
	}
}

func From(src []byte) Type {
	return classifyMime(fromBytesSniff(src))
}
