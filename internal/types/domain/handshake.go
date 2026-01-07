package domain

import (
	"github.com/labi-le/belphegor/internal/metadata"
)

type EventHandshake = Event[Handshake]

type Handshake struct {
	Version  string
	MetaData Device
	Port     uint32
}

func NewGreet(opts ...GreetOption) EventHandshake {
	greet := &Handshake{
		Version: metadata.Version,
	}

	for _, opt := range opts {
		opt(greet)
	}

	return NewEvent[Handshake](*greet)
}

type GreetOption func(g *Handshake)

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
