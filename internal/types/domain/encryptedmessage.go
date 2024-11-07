package domain

import (
	proto2 "github.com/labi-le/belphegor/internal/types/proto"
	"google.golang.org/protobuf/proto"
)

type EncryptedMessage struct {
	Message []byte
}

func (e EncryptedMessage) Proto() proto.Message {
	return &proto2.EncryptedMessage{
		Message: e.Message,
	}
}
