package node

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"

	"github.com/labi-le/belphegor/internal/metadata"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/ctxlog"
	"github.com/labi-le/belphegor/pkg/protoutil"
	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog"
)

var (
	ErrVersionMismatch = errors.New("nodes have major differences, handshake impossible")
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
			domain.WithMetadata(meta),
			domain.WithPort(uint16(port)),
		),
		private: privateKey,
		logger:  logger,
	}, nil
}

func (h *handshake) exchange(ctx context.Context, conn *quic.Conn, incoming bool) (domain.EventHandshake, error) {
	stream, err := openOrAcceptStream(ctx, conn, incoming)
	if err != nil {
		return domain.EventHandshake{}, fmt.Errorf("openOrAcceptStream error: %w", err)
	}
	defer stream.Close()

	if _, err := protoutil.EncodeWriter(h.my.Proto(), stream); err != nil {
		return domain.EventHandshake{}, fmt.Errorf("send greeting: %w", err)
	}

	from, err := domain.NewGreetFromReader(stream)
	if err != nil {
		return domain.EventHandshake{}, fmt.Errorf("receive greeting: %w", err)
	}

	ctxLog := ctxlog.Op(h.logger, "exchange")
	ctxLog.Trace().
		Str("node", from.Payload.MetaData.String()).
		Str("addr", conn.RemoteAddr().String()).
		Msg("received greeting")

	if metadata.IsMajorDifference(h.my.Payload.Version, from.Payload.Version) {
		ctxLog.Warn().
			Str("local", h.my.Payload.Version).
			Str("remote", from.Payload.Version).
			Msg("version mismatch")
		return domain.EventHandshake{}, ErrVersionMismatch
	}

	return from, nil
}
