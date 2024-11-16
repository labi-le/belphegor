package domain

import (
	"github.com/labi-le/belphegor/internal"
	"github.com/labi-le/belphegor/internal/types/proto"
	"github.com/labi-le/belphegor/pkg/protoutil"
	pb "google.golang.org/protobuf/proto"
	"io"
)

type Greet struct {
	Version   string
	MetaData  MetaData
	Port      uint32
	PublicKey []byte
}

func NewGreet(opts ...GreetOption) Greet {
	greet := &Greet{
		Version: internal.Version,
	}

	for _, opt := range opts {
		opt(greet)
	}

	return *greet
}

// GreetFromProto создает Greet из protobuf сообщения
func GreetFromProto(m *proto.GreetMessage) Greet {
	return Greet{
		Version:   m.Version,
		MetaData:  MetaDataFromProto(m.Device),
		Port:      m.Port,
		PublicKey: m.PublicKey,
	}
}

func NewGreetFromReader(reader io.Reader) (Greet, error) {
	var gp proto.GreetMessage
	if err := protoutil.DecodeReader(reader, &gp); err != nil {
		return Greet{}, err
	}

	return GreetFromProto(&gp), nil
}

func (g Greet) Proto() pb.Message {
	return &proto.GreetMessage{
		Version:   g.Version,
		Device:    g.MetaData.Proto(),
		Port:      g.Port,
		PublicKey: g.PublicKey,
	}
}

type GreetOption func(g *Greet)

func WithPublicKey(key []byte) GreetOption {
	return func(g *Greet) {
		g.PublicKey = key
	}
}

func WithMetadata(opt MetaData) GreetOption {
	return func(g *Greet) {
		g.MetaData = opt
	}
}
