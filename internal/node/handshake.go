package node

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/labi-le/belphegor/internal/metadata"
	"github.com/labi-le/belphegor/internal/protocol"
	"github.com/labi-le/belphegor/internal/transport"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/ctxlog"
	"github.com/rs/zerolog"
)

var (
	ErrVersionMismatch = errors.New("nodes have major differences, handshake impossible")
)

type handshake struct {
	my     domain.EventHandshake
	logger zerolog.Logger
}

func newHandshake(meta domain.Device, port int, logger zerolog.Logger) *handshake {
	return &handshake{
		my: domain.NewGreet(
			domain.WithMetadata(meta),
			domain.WithPort(uint16(port)),
		),
		logger: logger,
	}
}

func (h *handshake) exchange(ctx context.Context, conn transport.Connection, incoming bool) (domain.EventHandshake, error) {
	var empty domain.EventHandshake
	stream, err := openOrAcceptStream(ctx, conn, incoming)
	if err != nil {
		return empty, fmt.Errorf("openOrAcceptStream error: %w", err)
	}
	defer func(stream io.Closer) { _ = stream.Close() }(stream)

	if _, err := stream.Write(protocol.MustEncode(h.my)); err != nil {
		return empty, fmt.Errorf("send greeting: %w", err)
	}

	from, err := protocol.DecodeExpect[domain.EventHandshake](stream)
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
