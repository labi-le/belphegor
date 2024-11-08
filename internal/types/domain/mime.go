package domain

import "bytes"

var mimeTypeMap = map[string]MimeType{
	"image/png":  MimeTypeImage,
	"text/plain": MimeTypeText,
}

func parseMimeType(ct string) MimeType {
	if mime, ok := mimeTypeMap[ct]; ok {
		return mime
	}
	return MimeTypeText
}

type MimeType int32

const (
	MimeTypeText MimeType = iota
	MimeTypeImage
)

func GetMimeType(data []byte) string {
	switch {
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x89, 0x50, 0x4E, 0x47}): // PNG
		return "image/png"
	case len(data) >= 2 && bytes.Equal(data[:2], []byte{0xFF, 0xD8}): // JPEG
		return "image/jpeg"
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x47, 0x49, 0x46, 0x38}): // GIF
		return "image/gif"
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x25, 0x50, 0x44, 0x46}): // PDF
		return "application/pdf"
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x50, 0x4B, 0x03, 0x04}): // ZIP
		return "application/zip"
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x52, 0x61, 0x72, 0x21}): // RAR
		return "application/x-rar-compressed"
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x1F, 0x8B, 0x08, 0x00}): // GZIP
		return "application/gzip"
	case len(data) >= 2 && bytes.Equal(data[:2], []byte{0x42, 0x4D}): // BMP
		return "image/bmp"
	default:
		return "application/octet-stream"
	}
}

func MimeFromData(src []byte) MimeType {
	return parseMimeType(GetMimeType(src))
}
