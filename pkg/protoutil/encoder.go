package protoutil

import (
	"encoding/binary"
	"github.com/labi-le/belphegor/pkg/pool/byteslice"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"io"
)

const (
	Length = 4
)

// encode encodes the source interface and returns the encoded byte slice.
func encode(src proto.Message) []byte {
	encoded, err := proto.Marshal(src)
	if err != nil {
		log.Error().AnErr("encode", err).Msg("failed to encode clipboard data")
	}

	return encoded
}

// EncodeWriter encodes the source interface writes it to the destination io.Writer.
func EncodeWriter(src Proto[proto.Message], w io.Writer) (int, error) {
	encoded := encode(src.Proto())

	combined := byteslice.Get(Length + len(encoded))
	defer byteslice.Put(combined)

	binary.BigEndian.PutUint32(combined[:Length], uint32(len(encoded)))
	copy(combined[Length:], encoded)

	return w.Write(combined[:Length+len(encoded)])
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
	lenBytes := byteslice.Get(Length)
	defer byteslice.Put(lenBytes)

	if _, err := io.ReadFull(r, lenBytes); err != nil {
		return 0, err
	}
	return int(binary.BigEndian.Uint32(lenBytes)), nil
}

type Proto[T proto.Message] interface {
	Proto() T
}
