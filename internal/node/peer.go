package node

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"

	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/ctxlog"
	"github.com/labi-le/belphegor/pkg/encrypter"
)

type PeerOption func(*Peer)

func WithConn(conn net.Conn) PeerOption {
	return func(p *Peer) {
		p.conn = conn
		p.addr = conn.RemoteAddr().(*net.TCPAddr).AddrPort()
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

func AcquirePeer(opts ...PeerOption) *Peer {
	p := &Peer{}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

type Peer struct {
	conn       net.Conn
	addr       netip.AddrPort
	metaData   domain.Device
	channel    *Channel
	cipher     *encrypter.Cipher
	stringRepr string
}

func (p *Peer) Addr() netip.AddrPort { return p.addr }

func (p *Peer) MetaData() domain.Device { return p.metaData }

func (p *Peer) Conn() net.Conn { return p.conn }

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
	ctxLog := ctxlog.Op("peer.Receive")
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
			msg, err := domain.ReceiveMessage(p.Conn(), p.cipher)
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

				return res.err
			}

			p.channel.Send(res.msg)

			ctxLog.Trace().Msgf(
				"received %d from %s",
				res.msg.Payload.ID,
				p.String(),
			)
		}
	}
}
