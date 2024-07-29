package data

import (
	"encoding/binary"
	"errors"
	"github.com/labi-le/belphegor/pkg/pool/byteslice"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"io"
)

const (
	Length         = 4
	MaxMessageSize = 16 << 20
)

func encode(src proto.Message) ([]byte, error) {
	encoded, err := proto.Marshal(src)
	if err != nil {
		log.Error().AnErr("encode", err).Msg("failed to encode clipboard data")
		return nil, err
	}

	return encoded, nil
}

func EncodeWriter(src proto.Message, w io.Writer) (int, error) {
	encoded, err := encode(src)
	if err != nil {
		return 0, err
	}

	totalLength := Length + len(encoded)
	packet := byteslice.Get(totalLength)
	defer byteslice.Put(packet)

	binary.BigEndian.PutUint32(packet[:Length], uint32(len(encoded)))
	copy(packet[Length:], encoded)

	log.Debug().Msgf("sent %d bytes", len(encoded))

	return w.Write(packet)
}

func DecodeReader(r io.Reader, dst proto.Message) error {
	header := byteslice.Get(Length)
	defer byteslice.Put(header)

	if _, err := io.ReadFull(r, header); err != nil {
		return err
	}

	length := binary.BigEndian.Uint32(header)
	if length > MaxMessageSize {
		return errors.New("message too large")
	}

	data := byteslice.Get(int(length))
	defer byteslice.Put(data)

	if _, err := io.ReadFull(r, data); err != nil {
		return err
	}

	log.Debug().Msgf("received %d bytes", length)

	return proto.Unmarshal(data, dst)
}
