package peer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/labi-le/belphegor/internal/channel"
	"github.com/labi-le/belphegor/internal/protocol"
	"github.com/labi-le/belphegor/internal/transport"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/ctxlog"
	"github.com/labi-le/belphegor/pkg/id"
	"github.com/labi-le/belphegor/pkg/network"
	"github.com/rs/zerolog"
)

type Peer struct {
	conn       transport.Connection
	metaData   domain.Device
	channel    *channel.Channel
	stringRepr string
	logger     zerolog.Logger
	deadline   network.Deadline
}

func New(
	conn transport.Connection,
	metadata domain.Device,
	channel *channel.Channel,
	logger zerolog.Logger,
	dd network.Deadline,
) *Peer {
	return &Peer{
		conn:     conn,
		metaData: metadata,
		channel:  channel,
		logger:   logger.Hook(addNodeHook(id.MyID)),
		deadline: dd,
	}
}

func addNodeHook(nodeID id.Unique) zerolog.HookFunc {
	return func(e *zerolog.Event, level zerolog.Level, msg string) {
		e.Int64("node_id", nodeID)
	}
}

func (p *Peer) MetaData() domain.Device { return p.metaData }

func (p *Peer) Conn() transport.Connection { return p.conn }

func (p *Peer) Close() error {
	return p.conn.Close()
}

func (p *Peer) String() string {
	if p.stringRepr == "" {
		p.stringRepr = fmt.Sprintf(
			"%s -> %s",
			p.MetaData().Name,
			p.conn.RemoteAddr().String(),
		)
	}

	return p.stringRepr
}

func (p *Peer) Receive(ctx context.Context) error {
	ctxLog := ctxlog.Op(p.logger, "peer.Receive")
	defer ctxLog.
		Info().
		Str("node", p.String()).
		Msg("disconnected")

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			stream, err := p.conn.AcceptStream(ctx)
			if err != nil {
				if isConnClosed(err) {
					return nil
				}
				ctxLog.Error().Err(err).Msg("failed to accept stream")
				continue
			}

			if handleErr := p.handleStream(ctx, stream); handleErr != nil {
				ctxLog.Trace().Err(handleErr).Msg("failed to handle stream")
			}
		}
	}
}

func isConnClosed(err error) bool {
	if errors.Is(err, context.Canceled) {
		return true
	}

	if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
		return true
	}

	msg := err.Error()
	return strings.Contains(msg, "closed") || strings.Contains(msg, "application error 0x0")
}

func (p *Peer) WriteContext(ctx context.Context, meta, raw []byte) error {
	stream, err := p.conn.OpenStream(ctx)
	if err != nil {
		return fmt.Errorf("open stream: %w", err)
	}

	defer func(stream transport.Stream) {
		if err := stream.Close(); err != nil {
			p.logger.Trace().Err(err).Msg("failed to close writer stream")
		}
	}(stream)

	if err := network.SetWriteDeadline(stream, p.deadline); err != nil {
		return err
	}

	if _, err := stream.Write(meta); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	if raw != nil {
		if _, err := stream.Write(raw); err != nil {
			return fmt.Errorf("write: %w", err)
		}
	}

	return nil
}

func (p *Peer) handleStream(ctx context.Context, stream transport.Stream) error {
	defer stream.Close()

	if err := network.SetReadDeadline(stream, p.deadline); err != nil {
		return err
	}

	event, err := protocol.DecodeEvent(stream)
	if err != nil {
		return fmt.Errorf("decode event: %w", err)
	}

	switch payload := event.(type) {
	case domain.EventMessage:
		return p.handleMessage(payload, stream)

	case domain.EventAnnounce:
		p.channel.Announce(payload)
		p.logger.Trace().
			Int64("msg_id", payload.Payload.ID).
			Msg("received announce")
		return nil

	case domain.EventRequest:
		return p.handleRequest(ctx, p.channel.LastMsg(), payload)

	default:
		return fmt.Errorf("unknown payload type: %T", payload)
	}
}

func (p *Peer) handleMessage(
	msg domain.EventMessage,
	reader io.Reader,
) error {
	data := make([]byte, msg.Payload.ContentLength)

	if _, err := io.ReadFull(reader, data); err != nil {
		return fmt.Errorf("read raw data: %w", err)
	}

	msg.Payload.Data = data

	p.logger.Trace().
		Int64("msg_id", msg.Payload.ID).
		Msg("received message")

	p.channel.Send(msg)

	return nil
}

func (p *Peer) Request(ctx context.Context, messageID id.Unique) error {
	req := domain.NewRequest(messageID)

	bytes, err := protocol.Encode(req)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	p.logger.Trace().Int64("msg_id", messageID).Msg("sending request packet")

	return p.WriteContext(ctx, bytes, nil)
}

func (p *Peer) handleRequest(ctx context.Context, ev domain.EventMessage, req domain.EventRequest) error {
	logger := domain.MsgLogger(p.logger, ev.Payload.ID)
	logger.Trace().Msg("received request")

	if ev.Payload.ID != req.Payload.ID {
		return nil
	}

	logger.Trace().Msg("sending")

	meta := ev
	meta.Payload.Data = nil

	dst, err := protocol.Encode(meta)
	if err != nil {
		return fmt.Errorf("encode response: %w", err)
	}

	return p.WriteContext(ctx, dst, ev.Payload.Data)
}
