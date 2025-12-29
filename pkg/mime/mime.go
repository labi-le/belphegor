package mime

import (
	"bytes"
	"strings"
)

var (
	imageTypes = map[string]struct{}{
		"image/png":  {},
		"image/jpeg": {},
		"image/jpg":  {},
		"image/gif":  {},
		"image/bmp":  {},
		"image/webp": {},
	}

	textTypes = map[string]struct{}{
		"text/plain;charset=utf-8": {},
		"text/plain":               {},
		"utf8_string":              {},
		"text":                     {},
		"string":                   {},
	}

	supportedTypes map[string]struct{}
)

func init() {
	supportedTypes = make(map[string]struct{}, len(imageTypes)+len(textTypes))
	for k := range imageTypes {
		supportedTypes[k] = struct{}{}
	}
	for k := range textTypes {
		supportedTypes[k] = struct{}{}
	}
}

func SupportedTypes() map[string]struct{} {
	return supportedTypes
}

func IsImage(mimeType string) bool {
	_, ok := imageTypes[strings.ToLower(mimeType)]
	return ok
}

func IsText(mimeType string) bool {
	_, ok := textTypes[strings.ToLower(mimeType)]
	return ok
}

func IsSupported(mimeType string) bool {
	_, ok := supportedTypes[strings.ToLower(mimeType)]
	return ok
}

func Type(data []byte) string {
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
