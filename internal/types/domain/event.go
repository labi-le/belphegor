package domain

import (
	"time"

	"github.com/labi-le/belphegor/internal/types/proto"
	pb "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type payloadConstraint interface {
	Heartbeat | Message | Handshake
}

type Event[T payloadConstraint] struct {
	Type    Type
	From    UniqueID
	Created time.Time
	Payload T
}

func (e Event[T]) Equal(ev Event[T]) bool {
	return e.From == ev.From
}

func (e Event[T]) Proto() pb.Message {
	ev := &proto.Event{
		Type:    e.Type.Proto(),
		Created: timestamppb.New(e.Created),
	}

	payloadProto(e, ev)

	return ev
}

func payloadProto[T payloadConstraint](e Event[T], ev *proto.Event) {
	switch p := any(e.Payload).(type) {
	case Heartbeat:
		ev.Payload = &proto.Event_Heartbeat{
			Heartbeat: &proto.HeartbeatPayload{},
		}
	case Message:
		ev.Payload = &proto.Event_Update{
			Update: p.Proto().(*proto.Message),
		}
	case Handshake:
		ev.Payload = &proto.Event_Handshake{
			Handshake: p.Proto().(*proto.Handshake),
		}
	}
}

func NewEvent[concrete payloadConstraint](payload concrete) Event[concrete] {
	return Event[concrete]{
		Type:    typeByConstraint(payload),
		Created: time.Now(),
		Payload: payload,
	}
}

func typeByConstraint[concrete payloadConstraint](constraint concrete) Type {
	switch any(constraint).(type) {
	case Heartbeat:
		return TypeHeartbeat
	case Message:
		return TypeUpdate
	case Handshake:
		return TypeHandshake
	}

	return 0
}

type Type int32

func (t Type) Proto() proto.Type {
	switch t {
	case TypeHeartbeat:
		return proto.Type_HEARTBEAT
	case TypeUpdate:
		return proto.Type_UPDATE
	case TypeHandshake:
		return proto.Type_HANDSHAKE
	}

	return 0
}

const (
	TypeHeartbeat Type = iota
	TypeUpdate
	TypeHandshake
)

type Heartbeat struct {
}

func (h Heartbeat) Proto() pb.Message {
	return new(proto.HeartbeatPayload)
}
