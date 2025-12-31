package domain

import (
	"io"

	"github.com/labi-le/belphegor/internal/metadata"
	"github.com/labi-le/belphegor/internal/types/proto"
	"github.com/labi-le/belphegor/pkg/protoutil"
)

type EventHandshake = Event[Handshake]

type Handshake struct {
	Version   string
	MetaData  Device
	Port      uint32
	PublicKey []byte
	Provider  ClipboardProvider
}

func NewGreet(opts ...GreetOption) EventHandshake {
	greet := &Handshake{
		Version:  metadata.Version,
		Provider: CurrentClipboardProvider,
	}

	for _, opt := range opts {
		opt(greet)
	}

	return NewEvent[Handshake](*greet)
}

func GreetFromProto(m *proto.Event) EventHandshake {
	hs := m.Payload.(*proto.Event_Handshake).Handshake

	return EventHandshake{
		Created: m.Created.AsTime(),
		Payload: Handshake{
			Version:   hs.Version,
			MetaData:  MetaDataFromProto(hs.Device),
			Port:      hs.Port,
			PublicKey: hs.PublicKey,
			Provider:  ClipboardProvider(hs.Provider),
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

func (g Handshake) Proto() *proto.Handshake {
	return &proto.Handshake{
		Version:   g.Version,
		Device:    g.MetaData.Proto(),
		Port:      g.Port,
		PublicKey: g.PublicKey,
		Provider:  proto.Clipboard(g.Provider),
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

func WithPort(port uint16) GreetOption {
	return func(g *Handshake) {
		g.Port = uint32(port)
	}
}
