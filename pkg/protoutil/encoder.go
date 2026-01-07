package protoutil

import (
	"encoding/binary"
	"io"

	"github.com/labi-le/belphegor/pkg/pool/byteslice"
	"google.golang.org/protobuf/proto"
)

const (
	Length            = 4
	DefaultBufferSize = 2048
)

func EncodeBytes(src proto.Message) ([]byte, error) {
	target := make([]byte, Length, DefaultBufferSize)

	options := proto.MarshalOptions{
		UseCachedSize: true,
	}

	target, err := options.MarshalAppend(target, src)
	if err != nil {
		return nil, err
	}

	msgLen := len(target) - Length
	binary.BigEndian.PutUint32(target[:Length], uint32(msgLen))

	return target, nil
}

func DecodeReader(r io.Reader, dst proto.Message) error {
	length, err := dataLen(r)
	if err != nil {
		return err
	}

	data := byteslice.Get(length)
	defer byteslice.Put(data)

	if _, decodeErr := io.ReadFull(r, data); decodeErr != nil {
		return decodeErr
	}

	return proto.Unmarshal(data, dst)
}

func dataLen(r io.Reader) (int, error) {
	var header [Length]byte

	if _, err := io.ReadFull(r, header[:]); err != nil {
		return 0, err
	}

	return int(binary.BigEndian.Uint32(header[:])), nil
}

type Proto[T proto.Message] interface {
	Proto() T
}
