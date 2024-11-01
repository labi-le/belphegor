package node

import (
	"crypto"
	"errors"
	"fmt"
	"github.com/labi-le/belphegor/internal/node/data"
	"github.com/labi-le/belphegor/pkg/encrypter"
	"github.com/rs/zerolog/log"
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

func WithMetaData(meta data.MetaData) PeerOption {
	return func(p *Peer) {
		p.metaData = meta
	}
}

func WithLocalClipboard(updates data.Channel) PeerOption {
	return func(p *Peer) {
		p.localClipboard = updates
	}
}

func WithCipher(cipher *encrypter.Cipher) PeerOption {
	return func(p *Peer) {
		p.cipher = cipher
	}
}

func AcquirePeer(opts ...PeerOption) Peer {
	p := &Peer{}
	for _, opt := range opts {
		opt(p)
	}
	return *p
}

type Peer struct {
	conn           net.Conn
	addr           netip.AddrPort
	metaData       data.MetaData
	localClipboard data.Channel
	cipher         *encrypter.Cipher
}

func (p Peer) Addr() netip.AddrPort { return p.addr }

func (p Peer) MetaData() data.MetaData { return p.metaData }

func (p Peer) Conn() net.Conn { return p.conn }

func (p Peer) Updates() data.Channel { return p.localClipboard }

func (p Peer) Signer() crypto.Signer { return p.cipher }

func (p Peer) Close() error { return p.conn.Close() }

func (p Peer) String() string {
	return fmt.Sprintf(
		"%s -> %s",
		p.MetaData().String(),
		p.addr.String(),
	)
}

func (p Peer) Receive(last *data.LastMessage) {
	for {
		msg, err := data.ReceiveMessage(p.Conn(), p.cipher)
		if err != nil {
			p.handleReceiveError(err)
			break
		}

		last.Update <- msg
		p.localClipboard <- msg

		log.Debug().Msgf(
			"received %s from %s",
			msg.ID(),
			p.String(),
		)
	}

	log.Info().Msgf("%s disconnected", p.String())
}

func (p Peer) handleReceiveError(err error) {
	const op = "peer.handleReceiveError"

	var opErr *net.OpError
	if errors.As(err, &opErr) || errors.Is(err, io.EOF) {
		log.Trace().AnErr(op, opErr).Msg("connection closed")
		return
	}

	log.Error().AnErr(op, err).Msg("failed to receive message")
}
