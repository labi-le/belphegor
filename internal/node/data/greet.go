package data

import (
	"github.com/labi-le/belphegor/internal"
	"github.com/labi-le/belphegor/internal/types"
	"io"
)

type Greet struct {
	Version   string
	Device    MetaData
	Port      uint32
	PublicKey []byte
}

func NewGreet(metadata MetaData) Greet {
	return Greet{
		Version: internal.Version,
		Device:  metadata,
	}
}

// GreetFromProto создает Greet из protobuf сообщения
func GreetFromProto(m *types.GreetMessage) Greet {
	return Greet{
		Version:   m.Version,
		Device:    MetaDataFromProto(m.Device),
		Port:      m.Port,
		PublicKey: m.PublicKey,
	}
}

// NewGreetFromReader теперь использует FromProto
func NewGreetFromReader(reader io.Reader) (Greet, error) {
	var proto types.GreetMessage
	if err := DecodeReader(reader, &proto); err != nil {
		return Greet{}, err
	}

	return GreetFromProto(&proto), nil
}

func (g Greet) MetaData() MetaData {
	return g.Device
}

func (g Greet) ToProto() *types.GreetMessage {
	return &types.GreetMessage{
		Version:   g.Version,
		Device:    g.Device.ToProto(),
		Port:      g.Port,
		PublicKey: g.PublicKey,
	}
}
