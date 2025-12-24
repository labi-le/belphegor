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

func EncodeWriter(src proto.Message, w io.Writer) (int, error) {
	buf := byteslice.Get(DefaultBufferSize)
	defer byteslice.Put(buf)

	target := buf[:Length]

	var err error
	options := proto.MarshalOptions{}
	target, err = options.MarshalAppend(target, src)
	if err != nil {
		return 0, err
	}

	msgLen := len(target) - Length

	binary.BigEndian.PutUint32(target[:Length], uint32(msgLen))

	return w.Write(target)
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
