package node

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"net"

	"github.com/labi-le/belphegor/internal/metadata"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/ctxlog"
	"github.com/labi-le/belphegor/pkg/encrypter"
	"github.com/labi-le/belphegor/pkg/protoutil"
	"github.com/rs/zerolog"
)

var (
	ErrMajorDifference = errors.New("nodes have major differences, handshake impossible")
)

type handshake struct {
	my      domain.EventHandshake
	private crypto.Decrypter
	logger  zerolog.Logger
}

func newHandshake(bitSize int, meta domain.Device, port int, logger zerolog.Logger) (*handshake, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return nil, fmt.Errorf("generate key error: %w", err)
	}

	return &handshake{
		my: domain.NewGreet(
			domain.WithPublicKey(encrypter.PublicKey2Bytes(privateKey.Public())),
			domain.WithMetadata(meta),
			domain.WithPort(uint16(port)),
		),
		private: privateKey,
		logger:  logger,
	}, nil
}

func (h *handshake) exchange(conn net.Conn) (domain.EventHandshake, *encrypter.Cipher, error) {
	if _, err := protoutil.EncodeWriter(h.my.Proto(), conn); err != nil {
		return domain.EventHandshake{}, nil, fmt.Errorf("send greeting: %w", err)
	}

	from, err := domain.NewGreetFromReader(conn)
	if err != nil {
		return domain.EventHandshake{}, nil, fmt.Errorf("receive greeting: %w", err)
	}

	ctxLog := ctxlog.Op(h.logger, "handshake.exchangeGreetings")
	ctxLog.Trace().
		Str("node", from.Payload.MetaData.String()).
		Str("addr", conn.RemoteAddr().String()).
		Msg("received greeting")

	if metadata.IsMajorDifference(h.my.Payload.Version, from.Payload.Version) {
		ctxLog.Warn().
			Str("local", h.my.Payload.Version).
			Str("remote", from.Payload.Version).
			Msg("version mismatch")
		return domain.EventHandshake{}, nil, ErrMajorDifference
	}

	return from, encrypter.NewCipher(h.private, encrypter.Bytes2PublicKey(from.Payload.PublicKey)), nil
}
