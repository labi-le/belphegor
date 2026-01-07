package domain

import (
	"github.com/labi-le/belphegor/internal/types/proto"
)

type EventRequest = Event[Request]

type Request struct {
	ID MessageID
}

func NewRequest(id MessageID) Event[Request] {
	return NewEvent(Request{ID: id})
}

func (r Request) Proto() *proto.RequestMessage {
	return &proto.RequestMessage{ID: r.ID}
}
