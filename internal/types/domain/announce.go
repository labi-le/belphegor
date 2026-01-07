package domain

import (
	"github.com/labi-le/belphegor/pkg/id"
	"github.com/labi-le/belphegor/pkg/mime"
)

type EventAnnounce = Event[Announce]

type Announce struct {
	ID          id.Unique
	MimeType    mime.Type
	ContentHash uint64
}
