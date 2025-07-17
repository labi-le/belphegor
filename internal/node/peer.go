package node

import (
	"crypto"
	"errors"
	"fmt"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/ctxlog"
	"github.com/labi-le/belphegor/pkg/encrypter"
	"io"
	"net"
	"net/netip"
)

type PeerOption func(*Peer)

func WithConn(conn net.Conn) PeerOption {
	return func(p *Peer) {
		p.conn = conn
		p.addr = conn.RemoteAddr().(*net.TCPAddr).AddrPort()
	}
}

func WithMetaData(meta domain.MetaData) PeerOption {
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
	conn     net.Conn
	addr     netip.AddrPort
	metaData domain.MetaData
	channel  *Channel
	cipher   *encrypter.Cipher
}

func (p *Peer) Addr() netip.AddrPort { return p.addr }

func (p *Peer) MetaData() domain.MetaData { return p.metaData }

func (p *Peer) Conn() net.Conn { return p.conn }

func (p *Peer) Signer() crypto.Signer { return p.cipher }

func (p *Peer) Close() error { return p.conn.Close() }

func (p *Peer) String() string {
	return fmt.Sprintf(
		"%s -> %s",
		p.MetaData().String(),
		p.addr.String(),
	)
}

func (p *Peer) Receive() {
	ctxLog := ctxlog.Op("peer.Receive")

	for {
		msg, err := domain.ReceiveMessage(p.Conn(), p.cipher)
		if err != nil {
			var opErr *net.OpError
			if errors.As(err, &opErr) || errors.Is(err, io.EOF) {
				ctxLog.Trace().Err(opErr).Msg("connection closed")
				return
			}

			ctxLog.Err(err).Msg("failed to receive message")
			break
		}

		p.channel.Send(msg)

		ctxLog.Debug().Msgf(
			"received %d from %s",
			msg.ID(),
			p.String(),
		)
	}

	ctxLog.Info().Msgf("%s disconnected", p.String())
}
