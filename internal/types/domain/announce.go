package domain

import (
	"github.com/labi-le/belphegor/pkg/id"
	"github.com/labi-le/belphegor/pkg/mime"
	"github.com/rs/zerolog"
)

type EventAnnounce = Event[Announce]

type Announce struct {
	ID            MessageID
	MimeType      mime.Type
	ContentHash   uint64
	ContentLength uint64
}

func (an Announce) MarshalZerologObject(e *zerolog.Event) {
	e.Object("id", an.ID)
	e.Int64("node", id.Author(id.Unique(an.ID)))
	e.Stringer("mime", an.MimeType)
	e.Uint64("length", an.ContentLength)
	e.Uint64("hash", an.ContentHash)
}

func (an Announce) Zero() bool {
	return an.ID == 0 || an.ContentHash == 0
}

func (an Announce) Duplicate(other Announce) bool {
	if an.ID == other.ID {
		return true
	}

	return an.ContentHash != 0 && an.ContentHash == other.ContentHash
}
