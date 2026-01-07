package protocol

import (
	"fmt"
	"io"

	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/internal/types/proto"
	"github.com/labi-le/belphegor/pkg/protoutil"
)

func DecodeEvent(r io.Reader) (any, error) {
	var pb proto.Event
	if err := protoutil.DecodeReader(r, &pb); err != nil {
		return nil, err
	}

	switch p := pb.Payload.(type) {
	case *proto.Event_Message:
		return ToDomainMessage(&pb, p.Message, nil), nil
	case *proto.Event_Announce:
		return ToDomainAnnounce(&pb, p.Announce), nil
	case *proto.Event_Request:
		return ToDomainRequest(&pb, p.Request), nil
	case *proto.Event_Handshake:
		return ToDomainHandshake(&pb, p.Handshake), nil
	default:
		return nil, fmt.Errorf("unknown event type %T", p)
	}
}

func Encode(v any) ([]byte, error) {
	pb := MapToProto(v)
	if pb == nil {
		return nil, fmt.Errorf("unsupported type for encoding: %T", v)
	}
	return protoutil.EncodeBytes(pb)
}

func MustEncode(v any) []byte {
	encode, err := Encode(v)
	if err != nil {
		panic(err)
	}

	return encode
}

func DecodeExpect[T domain.AnyEvent](r io.Reader) (T, error) {
	var empty T
	event, err := DecodeEvent(r)
	if err != nil {
		return empty, err
	}

	typed, ok := event.(T)
	if !ok {
		return empty, fmt.Errorf("expected %T, got %T", empty, event)
	}

	return typed, nil
}
