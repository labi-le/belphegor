package node

import (
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/labi-le/belphegor/internal/node/data"
	"github.com/labi-le/belphegor/internal/types"
	"github.com/labi-le/belphegor/pkg/encrypter"
	"github.com/labi-le/belphegor/pkg/pool"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
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
	id *types.Device,
	updates Channel,
	cipher *encrypter.Cipher,
) *Peer {
	p := peerPool.Acquire()
	p.conn = conn
	p.addr = addr
	p.device = id
	p.localClipboard = updates
	p.cipher = cipher

	return p
}

type Peer struct {
	conn           net.Conn
	addr           netip.AddrPort
	device         *types.Device
	localClipboard Channel
	cipher         *encrypter.Cipher
}

func (p *Peer) Release() {
	_ = p.Close()

	p.device = nil
	p.addr = netip.AddrPort{}
	p.conn = nil
	p.localClipboard = nil

	peerPool.Release(p)
}

func (p *Peer) Addr() netip.AddrPort { return p.addr }

func (p *Peer) Device() *types.Device { return p.device }

func (p *Peer) Conn() net.Conn { return p.conn }

func (p *Peer) Updates() Channel { return p.localClipboard }

func (p *Peer) Close() error { return p.conn.Close() }

func (p *Peer) String() string {
	return fmt.Sprintf(
		"%s -> %s",
		prettyDevice(p.device),
		p.addr.String(),
	)
}

func prettyDevice(id *types.Device) string {
	return fmt.Sprintf(
		"%s (%s)",
		id.Name,
		id.UniqueID,
	)
}

func (p *Peer) Receive(last *data.LastMessage) {
	for {
		msg, err := p.receiveMessage()
		if err != nil {
			p.handleReceiveError(err)
			break
		}

		last.Update <- msg
		p.localClipboard <- msg

		log.Debug().Msgf(
			"received %s from %s",
			msg.Header.ID,
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

// receiveMessage receives a message from the node.
func (p *Peer) receiveMessage() (*data.Message, error) {
	var message types.Message

	var encrypt types.EncryptedMessage
	if decodeEnc := decodeReader(p.Conn(), &encrypt); decodeEnc != nil {
		return &data.Message{}, decodeEnc
	}

	decrypt, decErr := p.cipher.Decrypt(rand.Reader, encrypt.Message, nil)
	if decErr != nil {
		return &data.Message{}, decErr
	}

	return data.MessageFromProto(&message), proto.Unmarshal(decrypt, &message)
}
