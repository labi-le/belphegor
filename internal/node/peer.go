package node

import (
	"errors"
	"fmt"
	"github.com/labi-le/belphegor/internal/node/data"
	"github.com/labi-le/belphegor/pkg/pool"
	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog/log"
	"io"
	"net"
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

func AcquirePeer(conn quic.Connection, stream quic.Stream, meta *data.MetaData, updates data.Channel) *Peer {
	p := peerPool.Acquire()
	p.stream = stream
	p.conn = conn
	p.metaData = meta
	p.localClipboard = updates

	return p
}

type Peer struct {
	conn           quic.Connection
	stream         quic.Stream
	metaData       *data.MetaData
	localClipboard data.Channel
}

func (p *Peer) Release() {
	_ = p.Close()

	p.metaData = nil
	p.conn = nil
	p.localClipboard = nil

	peerPool.Release(p)
}

func (p *Peer) MetaData() *data.MetaData { return p.metaData }

func (p *Peer) Conn() quic.Connection { return p.conn }

func (p *Peer) Stream() quic.Stream { return p.stream }

func (p *Peer) Updates() data.Channel { return p.localClipboard }

func (p *Peer) Close() error { return p.conn.CloseWithError(ErrCodeNoError, "Close by peer") }

func (p *Peer) String() string {
	return fmt.Sprintf(
		"%s -> %s",
		p.MetaData().String(),
		p.Conn().RemoteAddr().String(),
	)
}

func (p *Peer) Receive(last *data.LastMessage) {
	for {
		msg, err := data.ReceiveMessage(p.stream)
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
}

// handleReceiveError handles errors when receiving data.
func (p *Peer) handleReceiveError(err error) {
	const op = "peer.handleReceiveError"

	var opErr *net.OpError
	if errors.As(err, &opErr) || errors.Is(err, io.EOF) {
		log.Trace().AnErr(op, opErr).Msg("connection closed")
		return
	}

	var appErr *quic.ApplicationError
	if errors.As(err, &appErr) && appErr.ErrorCode == ErrCodeNoError {
		log.Info().Msgf("%s disconnected", p.String())
		return
	}
	log.Error().AnErr(op, err).Msg("failed to receive message")
}
