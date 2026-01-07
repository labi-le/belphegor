package domain

import (
	"github.com/labi-le/belphegor/internal/types/proto"
	"github.com/labi-le/belphegor/pkg/id"
	"github.com/labi-le/belphegor/pkg/mime"
	"github.com/labi-le/belphegor/pkg/protoutil"
)

var _ protoutil.Proto[*proto.Announce] = Announce{}

type EventAnnounce = Event[Announce]

type Announce struct {
	ID          id.Unique
	MimeType    mime.Type
	ContentHash uint64
}

func (m Announce) Proto() *proto.Announce {
	return &proto.Announce{
		MimeType:    proto.Mime(m.MimeType),
		ID:          m.ID,
		ContentHash: m.ContentHash,
	}
}
func AnnounceFromProto(proto *proto.Event, payload *proto.Announce) EventAnnounce {
	return EventAnnounce{
		From:    id.Author(payload.GetID()),
		Created: proto.GetCreated().AsTime(),
		Payload: Announce{
			ID:          payload.GetID(),
			MimeType:    mime.Type(payload.GetMimeType()),
			ContentHash: payload.GetContentHash(),
		},
	}
}
