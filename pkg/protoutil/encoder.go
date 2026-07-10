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

// vtMarshaler and vtUnmarshaler are implemented by vtprotobuf-generated
// messages. When a message satisfies them the reflection-free generated codec
// is used; otherwise the standard reflective proto path is the fallback.
type (
	vtMarshaler interface {
		SizeVT() int
		MarshalToSizedBufferVT(data []byte) (int, error)
	}
	vtUnmarshaler interface {
		Reset()
		UnmarshalVT(data []byte) error
	}
)

func EncodeBytes(src proto.Message) ([]byte, error) {
	if vt, ok := src.(vtMarshaler); ok {
		size := vt.SizeVT()
		buf := make([]byte, Length+size)
		if _, err := vt.MarshalToSizedBufferVT(buf[Length:]); err != nil {
			return nil, err
		}
		binary.BigEndian.PutUint32(buf[:Length], uint32(size))
		return buf, nil
	}

	target := make([]byte, Length, DefaultBufferSize)
	options := proto.MarshalOptions{UseCachedSize: true}
	target, err := options.MarshalAppend(target, src)
	if err != nil {
		return nil, err
	}

	binary.BigEndian.PutUint32(target[:Length], uint32(len(target)-Length))
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

	if vt, ok := dst.(vtUnmarshaler); ok {
		vt.Reset() // UnmarshalVT merges; reset to match proto.Unmarshal semantics
		return vt.UnmarshalVT(data)
	}
	return proto.Unmarshal(data, dst)
}

func dataLen(r io.Reader) (int, error) {
	header := byteslice.Get(Length)
	defer byteslice.Put(header)

	if _, err := io.ReadFull(r, header); err != nil {
		return 0, err
	}

	return int(binary.BigEndian.Uint32(header)), nil
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

	if vt, ok := src.(vtMarshaler); ok {
		size := vt.SizeVT()
		need := Length + size

		buf := *bufPtr
		if cap(buf) < need {
			buf = make([]byte, need)
		} else {
			buf = buf[:need]
		}
		*bufPtr = buf

		if _, err := vt.MarshalToSizedBufferVT(buf[Length:need]); err != nil {
			return err
		}
		binary.BigEndian.PutUint32(buf[:Length], uint32(size))

		_, err := w.Write(buf)
		return err
	}

	buf := (*bufPtr)[:Length]
	options := proto.MarshalOptions{UseCachedSize: true}
	buf, err := options.MarshalAppend(buf, src)
	if err != nil {
		return err
	}
	*bufPtr = buf

	binary.BigEndian.PutUint32(buf[:Length], uint32(len(buf)-Length))
	_, err = w.Write(buf)
	return err
}
