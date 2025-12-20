package domain

import (
	"io"

	"github.com/labi-le/belphegor/internal"
	"github.com/labi-le/belphegor/internal/types/proto"
	"github.com/labi-le/belphegor/pkg/protoutil"
	pb "google.golang.org/protobuf/proto"
)

type EventHandshake = Event[Handshake]

type Handshake struct {
	Version   string
	MetaData  Device
	Port      uint32
	PublicKey []byte
}

func NewGreet(opts ...GreetOption) EventHandshake {
	greet := &Handshake{
		Version: internal.Version,
	}

	for _, opt := range opts {
		opt(greet)
	}

	return NewEvent[Handshake](*greet)
}

func GreetFromProto(m *proto.Event) EventHandshake {
	hs := m.Payload.(*proto.Event_Handshake).Handshake

	return EventHandshake{
		Type:    TypeHandshake,
		From:    hs.Device.ID,
		Created: m.Created.AsTime(),
		Payload: Handshake{
			Version:   hs.Version,
			MetaData:  MetaDataFromProto(hs.Device),
			Port:      hs.Port,
			PublicKey: hs.PublicKey,
		},
	}
}

func NewGreetFromReader(reader io.Reader) (EventHandshake, error) {
	var gp proto.Event
	if err := protoutil.DecodeReader(reader, &gp); err != nil {
		return EventHandshake{}, err
	}

	return GreetFromProto(&gp), nil
}

func (g Handshake) Proto() pb.Message {
	return &proto.Handshake{
		Version:   g.Version,
		Device:    g.MetaData.Proto(),
		Port:      g.Port,
		PublicKey: g.PublicKey,
	}
}

type GreetOption func(g *Handshake)

func WithPublicKey(key []byte) GreetOption {
	return func(g *Handshake) {
		g.PublicKey = key
	}
}

func WithMetadata(opt Device) GreetOption {
	return func(g *Handshake) {
		g.MetaData = opt
	}
}
