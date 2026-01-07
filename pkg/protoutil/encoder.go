package protoutil

import (
	"encoding/binary"
	"io"
	"sync"

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

var encodePool = sync.Pool{
	New: func() any {
		b := make([]byte, Length, DefaultBufferSize)
		return &b
	},
}

func EncodeToWriter(w io.Writer, src proto.Message) error {
	bufPtr := encodePool.Get().(*[]byte)
	defer encodePool.Put(bufPtr)

	buf := *bufPtr

	buf = buf[:Length]

	options := proto.MarshalOptions{
		UseCachedSize: true,
	}

	var err error
	buf, err = options.MarshalAppend(buf, src)
	if err != nil {
		return err
	}

	*bufPtr = buf

	msgLen := len(buf) - Length
	binary.BigEndian.PutUint32(buf[:Length], uint32(msgLen))

	_, err = w.Write(buf)
	return err
}
