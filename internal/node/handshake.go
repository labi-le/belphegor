package node

import (
	"context"
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
	my     domain.EventHandshake
	logger zerolog.Logger
}

func newHandshake(meta domain.Device, port int, logger zerolog.Logger) (*handshake, error) {
	return &handshake{
		my: domain.NewGreet(
			domain.WithMetadata(meta),
			domain.WithPort(uint16(port)),
		),
		logger: logger,
	}, nil
}

func (h *handshake) exchange(ctx context.Context, conn *quic.Conn, incoming bool) (domain.EventHandshake, error) {
	var empty domain.EventHandshake
	stream, err := openOrAcceptStream(ctx, conn, incoming)
	if err != nil {
		return empty, fmt.Errorf("openOrAcceptStream error: %w", err)
	}
	defer func(stream *quic.Stream) { _ = stream.Close() }(stream)

	if _, err := protoutil.EncodeWriter(h.my.Proto(), stream); err != nil {
		return empty, fmt.Errorf("send greeting: %w", err)
	}

	from, err := domain.NewGreetFromReader(stream)
	if err != nil {
		return empty, fmt.Errorf("receive greeting: %w", err)
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
		return empty, ErrVersionMismatch
	}

	return from, nil
}
