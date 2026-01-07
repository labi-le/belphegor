package domain

import (
	"time"

	"github.com/labi-le/belphegor/pkg/id"
)

type AnyEvent interface {
	isEvent()
}

type payloadConstraint interface {
	Handshake | Message | Announce | Request
}

type OwnerID = id.Unique

type Event[T payloadConstraint] struct {
	From    OwnerID
	Created time.Time
	Payload T
}

func (e Event[T]) isEvent() {}

func NewEvent[concrete payloadConstraint](payload concrete) Event[concrete] {
	return Event[concrete]{
		Created: time.Now(),
		Payload: payload,
		From:    id.MyID,
	}
}
