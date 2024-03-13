package belphegor

import (
	"belphegor/pkg/pool/byteslice"
	"encoding/binary"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"time"
)

const DataLength = 4

// encode encodes the source interface and returns the encoded byte slice.
func encode(src proto.Message) []byte {
	encoded, err := proto.Marshal(src)
	if err != nil {
		log.Error().Msgf("failed to encode clipboard data: %s", err)
	}

	return encoded
}

// encodeWriter encodes the source interface writes it to the destination io.Writer.
func encodeWriter(src proto.Message, w io.Writer) (int, error) {
	encoded := encode(src)

	lenBytes := byteslice.Get(DataLength)
	defer byteslice.Put(lenBytes)

	binary.BigEndian.PutUint32(lenBytes, uint32(len(encoded)))

	// Set write timeout if the writer implements net.Conn
	if conn, ok := w.(*net.TCPConn); ok {
		err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err != nil {
			return 0, err
		}
		defer conn.SetWriteDeadline(time.Time{}) // Reset the deadline when done
	}

	n, err := w.Write(lenBytes)
	if err != nil {
		return n, err
	}

	return w.Write(encoded)
}

func decodeReader(r io.Reader, dst proto.Message) error {
	length, err := dataLen(r)
	if err != nil {
		return err
	}

	data := byteslice.Get(length)
	defer byteslice.Put(data)

	if decodeErr := decodeBytes(r, data); decodeErr != nil {
		return decodeErr
	}

	return proto.Unmarshal(data, dst)
}

func dataLen(r io.Reader) (int, error) {
	lenBytes := byteslice.Get(DataLength)
	defer byteslice.Put(lenBytes)

	if _, err := io.ReadFull(r, lenBytes); err != nil {
		return 0, err
	}
	return int(binary.BigEndian.Uint32(lenBytes)), nil
}

func decodeBytes(r io.Reader, dst []byte) error {
	if _, err := io.ReadFull(r, dst); err != nil {
		return err
	}

	return nil
}
