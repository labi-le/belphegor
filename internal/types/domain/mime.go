package domain

import (
	"github.com/labi-le/belphegor/pkg/mime"
)

type MimeType int32

const (
	MimeTypeText MimeType = iota
	MimeTypeImage
)

var mimeTypeMap = map[string]MimeType{
	"image/png":  MimeTypeImage,
	"text/plain": MimeTypeText,
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
