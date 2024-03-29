package internal

import (
	"crypto/rand"
	"errors"
	gen "github.com/labi-le/belphegor/internal/types"
	"github.com/labi-le/belphegor/pkg/clipboard"
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
	id UniqueID,
	updates Channel,
	cipher *encrypter.Cipher,
) *Peer {
	p := peerPool.Acquire()
	p.conn = conn
	p.addr = addr
	p.id = id
	p.updates = updates
	p.received = &lastMessage{Message: AcquireMessage([]byte{})}
	p.cipher = cipher

	return p
}

type Peer struct {
	conn    net.Conn
	addr    netip.AddrPort
	id      UniqueID
	updates Channel

	received *lastMessage
	cipher   *encrypter.Cipher
}

func (p *Peer) Release() {
	_ = p.Close()

	p.id = ""
	p.addr = netip.AddrPort{}
	p.conn = nil
	p.updates = nil
	p.received = nil

	peerPool.Release(p)
}

func (p *Peer) Addr() netip.AddrPort {
	return p.addr
}

func (p *Peer) ID() UniqueID {
	return p.id
}

func (p *Peer) Conn() net.Conn {
	return p.conn
}

func (p *Peer) Updates() Channel {
	return p.updates
}

func (p *Peer) Close() error {
	return p.conn.Close()
}

func (p *Peer) String() string {
	return p.addr.String()
}

func (p *Peer) Receive(cm clipboard.Manager) {
	for {
		msg, err := p.receiveMessage()
		if err != nil {
			p.handleReceiveError(err)
			break
		}

		p.received.Set(msg)
		_ = cm.Set(msg.Data.Raw)
		p.updates.Write(msg.Data.Raw)

		log.Debug().Msgf(
			"received %s from %s by hashBytes %x",
			msg.Header.ID,
			p.ID(),
			shortHash(msg.Data.Hash),
		)
	}

	log.Info().Msgf("node %s disconnected", p.ID())
}

// handleReceiveError handles errors when receiving data.
func (p *Peer) handleReceiveError(err error) {
	if errors.Is(err, io.EOF) {
		log.Trace().Msg("connection closed by EOF (similar to invalid message)")
		return
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		log.Trace().Err(opErr).Msg("connection closed by OpError")
		return
	}

	log.Error().Err(err).Msg("failed to receive message")
}

// receiveMessage receives a message from the node.
func (p *Peer) receiveMessage() (*gen.Message, error) {
	var message gen.Message

	var encrypt gen.EncryptedMessage
	if decodeEnc := decodeReader(p.Conn(), &encrypt); decodeEnc != nil {
		return &message, decodeEnc
	}

	decrypt, decErr := p.cipher.Decrypt(rand.Reader, encrypt.Message, nil)
	if decErr != nil {
		return &message, decErr
	}

	return &message, proto.Unmarshal(decrypt, &message)

	//if err := p.cipher.DecryptReader(p.conn, &message); err != nil {
	//	return nil, err
	//}
	//
	//return &message, nil
}
