package domain

import (
	"github.com/labi-le/belphegor/pkg/mime"
)

type MimeType int32

func (t MimeType) IsImage() bool {
	return t == MimeTypeImage
}

const (
	MimeTypeText MimeType = iota
	MimeTypeImage
)

var mimeTypeMap = map[string]MimeType{
	"image/png":  MimeTypeImage,
	"image/jpeg": MimeTypeImage,
	"image/jpg":  MimeTypeImage,
	"image/gif":  MimeTypeImage,
	"image/bmp":  MimeTypeImage,
	"image/webp": MimeTypeImage,

	"text/plain":               MimeTypeText,
	"text/plain;charset=utf-8": MimeTypeText,
	"utf8_string":              MimeTypeText,
	"text":                     MimeTypeText,
	"string":                   MimeTypeText,
}

func parseMimeType(ct string) MimeType {
	if mimeType, ok := mimeTypeMap[ct]; ok {
		return mimeType
	}
	return MimeTypeText
}

func mimeFromData(src []byte) MimeType {
	return parseMimeType(mime.Type(src))
}
