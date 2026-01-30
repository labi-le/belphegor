package protocol

import (
	"sync"

	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/internal/types/proto"
	"github.com/labi-le/belphegor/pkg/id"
	"github.com/labi-le/belphegor/pkg/mime"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var eventProtoPool = sync.Pool{
	New: func() any { return &proto.Event{} },
}

func releaseEvent(pb *proto.Event) {
	pb.Reset()
	eventProtoPool.Put(pb)
}

func MapToProto(v any) *proto.Event {
	pb := eventProtoPool.Get().(*proto.Event)
	pb.Reset()

	switch e := v.(type) {
	case domain.EventMessage:
		pb.Created = timestamppb.New(e.Created)
		pb.Payload = &proto.Event_Message{
			Message: &proto.Message{
				ID:            e.Payload.ID.Int64(),
				ContentLength: e.Payload.ContentLength,
				MimeType:      toProtoMime(e.Payload.MimeType),
				ContentHash:   e.Payload.ContentHash,
				Name:          e.Payload.Name,
			},
		}
		return pb

	case domain.EventAnnounce:
		pb.Created = timestamppb.New(e.Created)
		pb.Payload = &proto.Event_Announce{
			Announce: &proto.Announce{
				ID:            e.Payload.ID.Int64(),
				MimeType:      toProtoMime(e.Payload.MimeType),
				ContentHash:   e.Payload.ContentHash,
				ContentLength: e.Payload.ContentLength,
			},
		}
		return pb

	case domain.EventRequest:
		pb.Created = timestamppb.New(e.Created)
		pb.Payload = &proto.Event_Request{
			Request: &proto.RequestMessage{
				ID: e.Payload.ID.Int64(),
			},
		}
		return pb

	case domain.EventHandshake:
		pb.Created = timestamppb.New(e.Created)
		pb.Payload = &proto.Event_Handshake{
			Handshake: &proto.Handshake{
				Version: e.Payload.Version,
				Port:    e.Payload.Port,
				Device: &proto.Device{
					Name: e.Payload.MetaData.Name,
					Arch: e.Payload.MetaData.Arch,
					ID:   e.Payload.MetaData.ID.Int64(),
				},
			},
		}
		return pb
	}

	releaseEvent(pb)
	return nil
}

func toDomainMessage(ev *proto.Event, msg *proto.Message, data []byte) domain.EventMessage {
	return domain.EventMessage{
		From:    domain.NodeID(id.Author(msg.GetID())),
		Created: ev.GetCreated().AsTime(),
		Payload: domain.Message{
			ID:            domain.MessageID(msg.GetID()),
			Data:          data,
			MimeType:      toDomainMime(msg.GetMimeType()),
			ContentHash:   msg.GetContentHash(),
			ContentLength: msg.GetContentLength(),
			Name:          msg.Name,
		},
	}
}

func toDomainAnnounce(ev *proto.Event, ann *proto.Announce) domain.EventAnnounce {
	return domain.EventAnnounce{
		From:    domain.NodeID(id.Author(ann.GetID())),
		Created: ev.GetCreated().AsTime(),
		Payload: domain.Announce{
			ID:            domain.MessageID(ann.GetID()),
			MimeType:      toDomainMime(ann.GetMimeType()),
			ContentHash:   ann.GetContentHash(),
			ContentLength: ann.GetContentLength(),
		},
	}
}

func toDomainRequest(ev *proto.Event, req *proto.RequestMessage) domain.EventRequest {
	return domain.EventRequest{
		From:    domain.NodeID(id.Author(req.GetID())),
		Created: ev.GetCreated().AsTime(),
		Payload: domain.Request{
			ID: domain.MessageID(req.GetID()),
		},
	}
}

func toDomainHandshake(ev *proto.Event, hs *proto.Handshake) domain.EventHandshake {
	return domain.EventHandshake{
		Created: ev.GetCreated().AsTime(),
		Payload: domain.Handshake{
			Version:  hs.GetVersion(),
			Port:     hs.GetPort(),
			MetaData: toDomainDevice(hs.GetDevice()),
		},
	}
}

func toDomainDevice(d *proto.Device) domain.Device {
	if d == nil {
		return domain.Device{Name: "unknown", Arch: "unknown"}
	}
	return domain.Device{
		ID:   domain.NodeID(d.GetID()),
		Name: d.GetName(),
		Arch: d.GetArch(),
	}
}

func toProtoMime(t mime.Type) proto.Mime {
	switch t {
	case mime.TypeText:
		return proto.Mime_TEXT
	case mime.TypeImage:
		return proto.Mime_IMAGE
	case mime.TypePath:
		return proto.Mime_PATH

	case mime.TypeAudio, mime.TypeVideo, mime.TypeBinary:
		return proto.Mime_PATH

	default:
		return proto.Mime_TEXT
	}
}

func toDomainMime(p proto.Mime) mime.Type {
	switch p {
	case proto.Mime_TEXT:
		return mime.TypeText
	case proto.Mime_IMAGE:
		return mime.TypeImage
	case proto.Mime_PATH:
		return mime.TypePath
	default:
		return mime.TypeText
	}
}
