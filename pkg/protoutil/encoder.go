package protoutil

import (
	"encoding/binary"
	"errors"
	"github.com/labi-le/belphegor/pkg/pool/byteslice"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"io"
)

// todo: make these values customizable
const (
	Length         = 4
	MaxMessageSize = 16 << 20
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

	lenBytes := byteslice.Get(Length)
	defer byteslice.Put(lenBytes)

	binary.BigEndian.PutUint32(lenBytes, uint32(len(encoded)))

	n, err := w.Write(lenBytes)
	if err != nil {
		return n, err
	}

	return w.Write(encoded)
}

func DecodeReader(r io.Reader, dst proto.Message) error {
	length, err := dataLen(r)
	if err != nil {
		return err
	}

	if length > MaxMessageSize {
		return errors.New("message too large")
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
