package node

import (
	"crypto"
	"errors"
	"fmt"
	"github.com/labi-le/belphegor/internal/node/data"
	"github.com/labi-le/belphegor/pkg/encrypter"
	"github.com/labi-le/belphegor/pkg/pool"
	"github.com/rs/zerolog/log"
	"io"
	"net"
	"net/netip"
)

var (
	peerPool = initPeerPool()
)

func initPeerPool() *pool.ObjectPool[*Peer] {
	p := pool.NewObjectPool[*Peer](10)
	p.New = func() *Peer {
		return &Peer{}
	}
	return p
}

func AcquirePeer(
	conn net.Conn,
	addr netip.AddrPort,
	meta *data.MetaData,
	updates data.Channel,
	cipher *encrypter.Cipher,
) *Peer {
	p := peerPool.Acquire()
	p.conn = conn
	p.addr = addr
	p.metaData = meta
	p.localClipboard = updates
	p.cipher = cipher

	return p
}

type Peer struct {
	conn           net.Conn
	addr           netip.AddrPort
	metaData       *data.MetaData
	localClipboard data.Channel
	cipher         *encrypter.Cipher
}

func (p *Peer) Release() {
	_ = p.Close()

	p.metaData = nil
	p.addr = netip.AddrPort{}
	p.conn = nil
	p.localClipboard = nil

	peerPool.Release(p)
}

func (p *Peer) Addr() netip.AddrPort { return p.addr }

func (p *Peer) MetaData() *data.MetaData { return p.metaData }

func (p *Peer) Conn() net.Conn { return p.conn }

func (p *Peer) Updates() data.Channel { return p.localClipboard }

func (p *Peer) Signer() crypto.Signer { return p.cipher }

func (p *Peer) Close() error { return p.conn.Close() }

func (p *Peer) String() string {
	return fmt.Sprintf(
		"%s -> %s",
		p.MetaData().String(),
		p.addr.String(),
	)
}

func (p *Peer) Receive(last *data.LastMessage) {
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

// handleReceiveError handles errors when receiving data.
func (p *Peer) handleReceiveError(err error) {
	const op = "peer.handleReceiveError"

	var opErr *net.OpError
	if errors.As(err, &opErr) || errors.Is(err, io.EOF) {
		log.Trace().AnErr(op, opErr).Msg("connection closed")
		return
	}

	log.Error().AnErr(op, err).Msg("failed to receive message")
}
