package peer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

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

	go func() {
		for range time.After(30 * time.Second) {
			stats := p.conn.ConnectionStats()
			ctxLog.Info().
				Int("packets_sent", int(stats.PacketsSent)).
				Int("packet_lost", int(stats.PacketsLost)).
				Int("packet_received", int(stats.PacketsReceived)).
				Int("bytes_lost", int(stats.BytesLost)).
				Int("bytes_sent", int(stats.BytesSent)).
				Dur("min_rtt", stats.MinRTT).
				Dur("latest_rtt", stats.LatestRTT).
				Send()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			ctxLog.Trace().Msg("waiting message")
			msg, err := p.receiveMessage(ctx)
			if err != nil {
				if isConnClosed(err) {
					return nil
				}

				return fmt.Errorf("error decoding: %w", err)
			}

			p.channel.Send(msg)

			ctxLog.Trace().Int(
				"msg_id",
				int(msg.Payload.ID),
			).Str("from", p.String()).Msg("received")
		}
	}
}

func isConnClosed(err error) bool {
	var err2 *quic.ApplicationError
	if errors.As(err, &err2) && err2.ErrorCode == ErrConnClosed {
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
		p.logger.Trace().Msg("close writer stream")

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

	if _, err := stream.Write(raw); err != nil {
		return fmt.Errorf("write: %w", err)
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
		return domain.EventMessage{}, err
	}

	var event proto.Event
	if decodeEnc := protoutil.DecodeReader(stream, &event); decodeEnc != nil {
		return empty, decodeEnc
	}

	payload, ok := event.GetPayload().(*proto.Event_Message)
	if !ok {
		return domain.EventMessage{}, fmt.Errorf("expected: %T, actual: %T", proto.Event_Message{}, event.GetPayload())
	}

	data := make([]byte, payload.Message.GetContentLength())
	if _, err := io.ReadFull(stream, data); err != nil {
		return empty, fmt.Errorf("read payload: %w", err)
	}

	return domain.FromProto(p.MetaData().ID, &event, payload, data), nil
}
