package peer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/labi-le/belphegor/internal/channel"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/internal/types/proto"
	"github.com/labi-le/belphegor/pkg/ctxlog"
	"github.com/labi-le/belphegor/pkg/id"
	"github.com/labi-le/belphegor/pkg/network"
	"github.com/labi-le/belphegor/pkg/protoutil"
	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog"
)

var (
	ErrConnClosed = quic.ApplicationErrorCode(0)
)

type Peer struct {
	conn       *quic.Conn
	metaData   domain.Device
	channel    *channel.Channel
	stringRepr string
	logger     zerolog.Logger
	deadline   network.Deadline
}

func New(
	conn *quic.Conn,
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

func (p *Peer) Conn() *quic.Conn { return p.conn }

func (p *Peer) Close() error {
	return p.conn.CloseWithError(ErrConnClosed, "closed conn")
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
			stream, err := p.Conn().AcceptUniStream(ctx)
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

	var err2 *quic.ApplicationError
	if errors.As(err, &err2) && err2.ErrorCode == ErrConnClosed {
		return true
	}

	var err3 *quic.IdleTimeoutError
	if errors.As(err, &err3) {
		return true
	}

	var opErr *net.OpError
	return errors.As(err, &opErr) || errors.Is(err, io.EOF)
}

func (p *Peer) WriteContext(ctx context.Context, meta, raw []byte) error {
	stream, err := p.conn.OpenUniStreamSync(ctx)
	if err != nil {
		return fmt.Errorf("open stream: %w", err)
	}
	defer func(stream *quic.SendStream) {
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

func (p *Peer) receiveMessage(ctx context.Context) (domain.EventMessage, error) {
	var empty domain.EventMessage

	stream, err := p.Conn().AcceptUniStream(ctx)
	if err != nil {
		return empty, fmt.Errorf("receive error: %w", err)
	}

	if err := network.SetReadDeadline(stream, p.deadline); err != nil {
		return empty, err
	}

	var event proto.Event
	if decodeEnc := protoutil.DecodeReader(stream, &event); decodeEnc != nil {
		return empty, decodeEnc
	}

	payload, ok := event.GetPayload().(*proto.Event_Message)
	if !ok {
		return empty, fmt.Errorf("expected: %T, actual: %T", proto.Event_Message{}, event.GetPayload())
	}

	data := make([]byte, payload.Message.GetContentLength())
	if _, err := io.ReadFull(stream, data); err != nil {
		return empty, fmt.Errorf("read payload: %w", err)
	}

	return domain.MessageFromProto(&event, payload.Message, data), nil
}

func (p *Peer) handleStream(ctx context.Context, stream *quic.ReceiveStream) error {
	defer stream.CancelRead(0)

	if err := network.SetReadDeadline(stream, p.deadline); err != nil {
		return err
	}

	var event proto.Event
	if err := protoutil.DecodeReader(stream, &event); err != nil {
		return fmt.Errorf("decode header: %w", err)
	}

	switch payload := event.GetPayload().(type) {
	case *proto.Event_Message:
		return p.handleMessage(&event, payload.Message, stream)

	case *proto.Event_Announce:
		return p.handleAnnounce(&event, payload.Announce)

	case *proto.Event_Request:
		return p.handleRequest(ctx, p.channel.LastMsg(), payload.Request)

	default:
		return fmt.Errorf("unknown payload type: %T", payload)
	}

}

func (p *Peer) handleMessage(
	header *proto.Event,
	meta *proto.Message,
	reader io.Reader,
) error {
	data := make([]byte, meta.GetContentLength())

	if _, err := io.ReadFull(reader, data); err != nil {
		return fmt.Errorf("read raw data: %w", err)
	}

	domainMsg := domain.MessageFromProto(header, meta, data)

	p.logger.Trace().
		Int64("msg_id", domainMsg.Payload.ID).
		Msg("received message")

	p.channel.Send(domainMsg)

	return nil
}

func (p *Peer) handleAnnounce(
	header *proto.Event,
	announce *proto.Announce,
) error {
	domainAnnounce := domain.AnnounceFromProto(header, announce)

	p.logger.Trace().
		Int64("msg_id", domainAnnounce.Payload.ID).
		Msg("received announce")

	p.channel.Announce(domainAnnounce)

	return nil
}

func (p *Peer) Request(ctx context.Context, messageID id.Unique) error {
	req := domain.NewRequest(messageID)

	bytes, err := protoutil.EncodeBytes(req.Proto())
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	p.logger.Trace().Int64("msg_id", messageID).Msg("sending request packet")

	return p.WriteContext(ctx, bytes, nil)
}

func (p *Peer) handleRequest(ctx context.Context, ev domain.EventMessage, request *proto.RequestMessage) error {
	logger := domain.MsgLogger(p.logger, ev.Payload.ID)
	logger.Trace().Msg("received request")

	if ev.Payload.ID != request.GetID() {
		return nil
	}

	logger.Trace().Msg("sending")

	meta := ev
	meta.Payload.Data = nil

	pt := meta.Proto()
	dst, _ := protoutil.EncodeBytes(pt)

	return p.WriteContext(ctx, dst, ev.Payload.Data)
}
