package node

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/ctxlog"
	"github.com/labi-le/belphegor/pkg/encrypter"
	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog"
)

type PeerOption func(*Peer)

func WithStream(conn *quic.Stream) PeerOption {
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

func WithCipher(cipher *encrypter.Cipher) PeerOption {
	return func(p *Peer) {
		p.cipher = cipher
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
	conn       *quic.Stream
	addr       net.Addr
	metaData   domain.Device
	channel    *Channel
	cipher     *encrypter.Cipher
	stringRepr string
	logger     zerolog.Logger
}

func (p *Peer) MetaData() domain.Device { return p.metaData }

func (p *Peer) Stream() *quic.Stream { return p.conn }

func (p *Peer) Signer() crypto.Signer { return p.cipher }

func (p *Peer) Close() error { return p.conn.Close() }

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
			msg, err := ReceiveMessage(p.Stream(), p.cipher, p.MetaData())
			resultChan <- readResult{msg: msg, err: err}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return p.Close()
		case res := <-resultChan:
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
