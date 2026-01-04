package domain

import (
	"time"

	"github.com/labi-le/belphegor/internal/types/proto"
	"github.com/labi-le/belphegor/pkg/id"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type payloadConstraint interface {
	Heartbeat | Handshake | Message
}

type OwnerID = id.Unique

type Event[T payloadConstraint] struct {
	From    OwnerID
	Created time.Time
	Payload T
}

func (e Event[T]) Proto() *proto.Event {
	ev := &proto.Event{
		Created: timestamppb.New(e.Created),
	}

	payloadProto(e, ev)

	return ev
}

func payloadProto[T payloadConstraint](e Event[T], ev *proto.Event) {
	switch p := any(e.Payload).(type) {
	case Heartbeat:
		ev.Payload = &proto.Event_Heartbeat{
			Heartbeat: p.Proto(),
		}

	case Message:
		ev.Payload = &proto.Event_Message{
			Message: p.Proto(),
		}

	case Handshake:
		ev.Payload = &proto.Event_Handshake{
			Handshake: p.Proto(),
		}
	}
}

func NewEvent[concrete payloadConstraint](payload concrete) Event[concrete] {
	return Event[concrete]{
		Created: time.Now(),
		Payload: payload,
	}
}

type Heartbeat struct{}

func (h Heartbeat) Proto() *proto.HeartbeatPayload {
	return new(proto.HeartbeatPayload)
}
