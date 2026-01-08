package domain

import (
	"github.com/labi-le/belphegor/pkg/id"
	"github.com/labi-le/belphegor/pkg/mime"
	"github.com/rs/zerolog"
)

type EventAnnounce = Event[Announce]

type Announce struct {
	ID            id.Unique
	MimeType      mime.Type
	ContentHash   uint64
	ContentLength uint64
}

func (m Announce) MarshalZerologObject(e *zerolog.Event) {
	e.Int64("id", m.ID)
	e.Int64("node", id.Author(m.ID))
	e.Stringer("mime", m.MimeType)
	e.Uint64("length", m.ContentLength)
	e.Uint64("hash", m.ContentHash)
}
