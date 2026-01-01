package node

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/ctxlog"
	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog"
)

type PeerOption func(*Peer)

func WithConn(conn *quic.Conn) PeerOption {
	return func(p *Peer) {
		p.conn = conn
	}
}

func WithPeerLogger(logger zerolog.Logger) PeerOption {
	return func(p *Peer) {
		p.logger = logger
	}
}

func WithAddr(addr net.Addr) PeerOption {
	return func(p *Peer) {
		p.addr = addr
	}
}

func WithMetaData(meta domain.Device) PeerOption {
	return func(p *Peer) {
		p.metaData = meta
	}
}

func WithChannel(updates *Channel) PeerOption {
	return func(p *Peer) {
		p.channel = updates
	}
}

func NewPeer(opts ...PeerOption) *Peer {
	p := new(Peer)
	for _, opt := range opts {
		opt(p)
	}
	return p
}

type Peer struct {
	conn       *quic.Conn
	addr       net.Addr
	metaData   domain.Device
	channel    *Channel
	stringRepr string
	logger     zerolog.Logger
}

func (p *Peer) MetaData() domain.Device { return p.metaData }

func (p *Peer) Conn() *quic.Conn { return p.conn }

func (p *Peer) Close() error {
	return p.conn.CloseWithError(quic.ApplicationErrorCode(0), "closed conn")
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
			stream, err := p.Conn().AcceptStream(ctx)
			if err != nil {
				close(resultChan)
				return
			}
			msg, err := ReceiveMessage(stream, p.MetaData())
			resultChan <- readResult{msg: msg, err: err}
			if err := stream.Close(); err != nil {
				return
			}
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

func (p *Peer) Write(data []byte) (int, error) {
	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	stream, err := p.conn.OpenStreamSync(ctx)
	if err != nil {
		return 0, fmt.Errorf("open stream: %w", err)
	}
	defer stream.Close()

	if err := stream.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
		return 0, fmt.Errorf("set write deadline: %w", err)
	}
	defer stream.SetWriteDeadline(time.Time{})

	if _, err := stream.Write(data); err != nil {
		return 0, fmt.Errorf("write: %w", err)
	}

	if err := stream.Close(); err != nil {
		return 0, fmt.Errorf("close stream: %w", err)
	}

	return 0, nil
}
