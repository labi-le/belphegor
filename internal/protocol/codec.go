package protocol

import (
	"fmt"
	"io"

	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/internal/types/proto"
	"github.com/labi-le/belphegor/pkg/protoutil"
)

func DecodeEvent(r io.Reader) (any, error) {
	pb, _ := eventProtoPool.Get().(*proto.Event)
	defer releaseEvent(pb)

	if err := protoutil.DecodeReader(r, pb); err != nil {
		return nil, fmt.Errorf("decode event: %w", err)
	}

	switch p := pb.GetPayload().(type) {
	case *proto.Event_Message:
		return toDomainMessage(pb, p.Message, nil), nil
	case *proto.Event_Announce:
		return toDomainAnnounce(pb, p.Announce), nil
	case *proto.Event_Request:
		return toDomainRequest(pb, p.Request), nil
	case *proto.Event_Handshake:
		return toDomainHandshake(pb, p.Handshake), nil
	default:
		return nil, fmt.Errorf("unknown event type %T", p)
	}
}

func Encode(v any) ([]byte, error) {
	pb := MapToProto(v)
	if pb == nil {
		return nil, fmt.Errorf("unsupported type for encoding: %T", v)
	}
	defer releaseEvent(pb)

	b, err := protoutil.EncodeBytes(pb)
	if err != nil {
		return nil, fmt.Errorf("encode event: %w", err)
	}
	return b, nil
}

func MustEncode(v any) []byte {
	encode, err := Encode(v)
	if err != nil {
		panic(err)
	}

	return encode
}

func DecodeExpect[T domain.AnyEvent](r io.Reader) (T, error) { //nolint:ireturn // generic decoder returns the caller-specified event type T
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

func WriteEvent(w io.Writer, v any) error {
	pb := MapToProto(v)
	if pb == nil {
		return fmt.Errorf("unsupported type for encoding: %T", v)
	}

	defer releaseEvent(pb)

	if err := protoutil.EncodeToWriter(w, pb); err != nil {
		return fmt.Errorf("write event: %w", err)
	}
	return nil
}
