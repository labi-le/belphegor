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
	addr       net.Addr
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
		addr:     conn.RemoteAddr(),
		metaData: metadata,
		channel:  channel,
		logger:   logger,
		deadline: dd,
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
			p.addr.String(),
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

	type readResult struct {
		msg domain.EventMessage
		err error
	}
	resultChan := make(chan readResult, 1)

	go func() {
		for {
			msg, err := p.receiveMessage(ctx)
			if err != nil {
				close(resultChan)
				return
			}
			resultChan <- readResult{msg: msg, err: err}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return p.Close()
		case res, ok := <-resultChan:
			if !ok {
				return nil
			}
			if res.err != nil {
				var opErr *net.OpError
				if errors.As(res.err, &opErr) || errors.Is(res.err, io.EOF) {
					ctxLog.Trace().Err(opErr).Msg("connection closed")
					return nil
				}

				return fmt.Errorf("error decoding: %w", res.err)
			}

			p.channel.Send(res.msg)

			ctxLog.Trace().Int(
				"msg_id",
				int(res.msg.Payload.ID),
			).Str("from", p.String()).Msg("received")
		}
	}
}

func (p *Peer) WriteContext(ctx context.Context, meta, raw []byte) error {
	stream, err := p.conn.OpenStreamSync(ctx)
	if err != nil {
		return fmt.Errorf("open stream: %w", err)
	}
	defer func(stream *quic.Stream) { _ = stream.Close() }(stream)

	if err := network.SetDeadline(stream, p.deadline); err != nil {
		return err
	}

	if _, err := stream.Write(meta); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	if _, err := stream.Write(raw); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	if err := stream.Close(); err != nil {
		return fmt.Errorf("close stream: %w", err)
	}

	return nil
}

func (p *Peer) receiveMessage(ctx context.Context) (domain.EventMessage, error) {
	var empty domain.EventMessage

	stream, err := p.Conn().AcceptStream(ctx)
	if err != nil {
		return empty, fmt.Errorf("receive error: %w", err)
	}

	defer func(stream *quic.Stream) { _ = stream.Close() }(stream)

	if err := network.SetDeadline(stream, p.deadline); err != nil {
		return domain.EventMessage{}, err
	}

	var event proto.Event
	if decodeEnc := protoutil.DecodeReader(stream, &event); decodeEnc != nil {
		return empty, decodeEnc
	}

	payload, ok := event.GetPayload().(*proto.Event_Message)
	if ok == false {
		return domain.EventMessage{}, fmt.Errorf("expected: %T, actual: %T", proto.Event_Message{}, event.GetPayload())
	}

	data := make([]byte, payload.Message.GetContentLength())
	if _, err := io.ReadFull(stream, data); err != nil {
		return empty, fmt.Errorf("read payload: %w", err)
	}

	return domain.FromProto(p.MetaData().ID, &event, payload, data), nil
}
