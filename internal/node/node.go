package node

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"

	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/internal/types/proto"
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/ctxlog"
	"github.com/labi-le/belphegor/pkg/encrypter"
	"github.com/labi-le/belphegor/pkg/id"
	"github.com/labi-le/belphegor/pkg/protoutil"
	"github.com/quic-go/quic-go"
)

var (
	ErrAlreadyConnected = errors.New("already connected")
)

type Node struct {
	clipboard eventful.Eventful
	peers     *Storage
	channel   *Channel
	opts      Options
}

// New creates a new instance of Node with the specified settings
func New(
	clipboard eventful.Eventful,
	peers *Storage,
	channel *Channel,
	opts ...Option,
) *Node {
	options := NewOptions(opts...)

	return &Node{
		clipboard: clipboard,
		peers:     peers,
		channel:   channel,
		opts:      options,
	}
}

// ConnectTo establishes a TCP connection to a remote clipboard at the specified address
func (n *Node) ConnectTo(ctx context.Context, addr string) error {
	ctxLog := ctxlog.Op(n.opts.Logger, "node.ConnectTo")

	config, err2 := generateTLSConfig()
	if err2 != nil {
		return fmt.Errorf("generateTLSConfig: %w", err2)
	}

	conn, err := quic.DialAddr(ctx, addr, config, generateQuicConfig(n.opts.KeepAlive))
	if err != nil {
		ctxLog.Error().AnErr("net.Dial", err).Msg("failed to handle connection")
		return err
	}

	if connErr := n.handleConnection(ctx, conn, false); connErr != nil {
		ctxLog.Error().AnErr("node.handleConnection", connErr).Msg("failed to handle connection")
		return connErr
	}

	return nil
}

func (n *Node) addPeer(hisHand domain.Handshake, cipher *encrypter.Cipher, stream *quic.Stream, addr net.Addr) (*Peer, error) {
	ctxLog := ctxlog.Op(n.opts.Logger, "node.addPeer")

	metadata := hisHand.MetaData
	if n.peers.Exist(metadata.UniqueID()) {
		ctxLog.Trace().Msgf("%s already connected, ignoring", metadata.String())
		return nil, ErrAlreadyConnected
	}

	peer := NewPeer(
		WithStream(stream),
		WithAddr(addr),
		WithMetaData(metadata),
		WithChannel(n.channel),
		WithCipher(cipher),
		WithPeerLogger(n.opts.Logger),
	)

	n.peers.Add(
		metadata.UniqueID(),
		peer,
	)
	return peer, nil
}

// Start starts the node by listening for incoming connections on the specified public port
func (n *Node) Start(ctx context.Context) error {
	defer n.channel.Close()

	ctxLog := ctxlog.Op(n.opts.Logger, "node.Start")

	config, err2 := generateTLSConfig()
	if err2 != nil {
		return fmt.Errorf("generateTLSConfig: %w", err2)
	}
	l, err := quic.ListenAddr(fmt.Sprintf(":%d", n.opts.PublicPort), config, generateQuicConfig(n.opts.KeepAlive))
	if err != nil {
		ctxLog.Err(err).Msg("failed to listen")
		return fmt.Errorf("node.Start: %w", err)
	}
	defer l.Close()

	go func() {
		<-ctx.Done()
		l.Close()
	}()

	addr := l.Addr().String()
	n.Notify("started on %s", addr)
	ctxLog.Info().
		Str("address", addr).
		Str("metadata", n.opts.Metadata.String()).
		Msg("started")

	go n.MonitorBuffer(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil

		default:
			conn, netErr := l.Accept(ctx)
			if netErr != nil {
				if errors.Is(netErr, net.ErrClosed) {
					break
				}

				if errors.Is(netErr, context.Canceled) {
					continue
				}

				ctxLog.
					Fatal().
					Err(netErr).
					Msg("failed to accept connection")
				return fmt.Errorf("node.Start: %w", netErr)
			}

			ctxLog.
				Trace().
				Msgf("accepted connection from %s", conn.RemoteAddr())

			go func() {
				if connErr := n.handleConnection(ctx, conn, true); connErr != nil {
					ctxLog.
						Err(connErr).
						Msg("failed to handle connection")
				}
			}()
		}
	}
}

func (n *Node) handleConnection(ctx context.Context, conn *quic.Conn, accept bool) error {
	ctxLog := ctxlog.Op(n.opts.Logger, "node.handleConnection").
		With().
		Str("node", n.Metadata().String()).
		Logger()

	hs, cipherErr := newHandshake(n.opts.BitSize, n.Metadata(), n.opts.PublicPort, n.opts.Logger)
	if cipherErr != nil {
		return cipherErr
	}

	stream, err := openOrAcceptStream(ctx, conn, accept)
	if err != nil {
		return fmt.Errorf("openOrAcceptStream error: %w", err)
	}

	hisHand, cipher, greetErr := hs.exchange(stream, conn.RemoteAddr())
	if greetErr != nil {
		if errors.Is(greetErr, ErrVersionMismatch) {
			return nil
		}

		return greetErr
	}

	peer, addErr := n.addPeer(
		hisHand.Payload,
		cipher,
		stream,
		conn.RemoteAddr(),
	)
	if addErr != nil {
		if errors.Is(addErr, ErrAlreadyConnected) {
			return nil
		}
		ctxLog.
			Err(addErr).
			Msg("failed to add")
		return addErr
	}
	defer n.peers.Delete(peer.MetaData().UniqueID())

	n.Notify("connected to %s", peer.MetaData().Name)
	defer n.Notify("Node disconnected %s", peer.MetaData().Name)

	ctxLog.Info().Msg("connected")

	return peer.Receive(ctx)
}

func openOrAcceptStream(ctx context.Context, conn *quic.Conn, accept bool) (*quic.Stream, error) {
	if accept {
		return conn.OpenStreamSync(ctx)
	}
	return conn.AcceptStream(ctx)
}

func (n *Node) Broadcast(msg domain.EventMessage) {
	ctxLog := ctxlog.Op(n.opts.Logger, "node.Broadcast").
		With().
		Int64("msg_id", msg.Payload.ID).
		Logger()

	n.peers.Tap(func(id id.Unique, peer *Peer) bool {
		ctx := ctxLog.
			With().
			Str("node", peer.String()).
			Logger()

		if id == msg.From {
			return true
		}

		ctx.Trace().Msg("sent")

		err := peer.Stream().SetWriteDeadline(time.Now().Add(n.opts.WriteTimeout))
		if err != nil {
			ctx.Trace().
				AnErr("SetWriteDeadline", err).
				Msg("cannot set write deadline")
			return true
		}
		defer peer.Stream().SetWriteDeadline(time.Time{})

		//_, encErr := WriteEncryptedMessage(msg, peer.Signer(), peer.Stream())
		_, encodeErr := protoutil.EncodeWriter(msg.Proto(), peer.conn)
		if encodeErr != nil {
			if errors.Is(encodeErr, net.ErrClosed) ||
				strings.Contains(encodeErr.Error(), "bad file descriptor") ||
				strings.Contains(encodeErr.Error(), "use of closed network connection") {

				ctx.Trace().Msg("connection closed during broadcast, removing peer")
			} else {
				ctx.Trace().
					AnErr("protoutil.EncodeWriter", encodeErr).
					Msg("failed to write encrypted message")
			}

			n.peers.Delete(peer.MetaData().UniqueID())
		}

		return true
	})
}

// MonitorBuffer starts monitoring the clipboard and subsequently sending data to other nodes
func (n *Node) MonitorBuffer(ctx context.Context) error {
	ctxLog := ctxlog.Op(n.opts.Logger, "node.MonitorBuffer")

	updates, watchErr := make(chan eventful.Update), make(chan error, 1)
	go func() {
		if err := n.clipboard.Watch(ctx, updates); err != nil {
			watchErr <- err
		}
	}()

	go func() {
		var (
			current domain.Message
		)
		for update := range updates {
			msg := domain.FromUpdate(update)
			if !msg.Duplicate(current) {
				ctxLog.
					Trace().
					Int64("msg_id", msg.ID).
					Msg("clipboard changed")

				current = msg
				n.channel.Send(current.Event(n.Metadata().UniqueID()))
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-watchErr:
			if err != nil {
				return fmt.Errorf("node.MonitorBuffer: %w", err)
			}
			return nil
		case msg := <-n.channel.Listen():
			if msg.From != n.opts.Metadata.UniqueID() {
				ctxLog.Trace().Int64("msg_id", msg.Payload.ID).Msg("set clipboard data")

				if _, err := n.clipboard.Write(msg.Payload.Data); err != nil {
					ctxLog.Error().Err(err).Send()
				}
			}

			go n.Broadcast(msg)
		}
	}
}

func (n *Node) Notify(message string, v ...any) {
	n.opts.Notifier.Notify(message, v...)
}

func (n *Node) Metadata() domain.Device {
	return n.opts.Metadata
}

func ReceiveMessage(conn *quic.Stream, decrypter crypto.Decrypter, data domain.Device) (domain.EventMessage, error) {
	var event proto.Event
	if decodeEnc := protoutil.DecodeReader(conn, &event); decodeEnc != nil {
		return domain.EventMessage{}, decodeEnc
	}

	payload, ok := event.Payload.(*proto.Event_Message)
	if ok == false {
		return domain.EventMessage{}, fmt.Errorf("expected: %T, actual: %T", proto.Event_Message{}, event.Payload)
	}

	return domain.FromProto(data.ID, &event, payload), nil

}

func generateTLSConfig() (*tls.Config, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("ecdsa.GenerateKey: %w", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("rand.Int: %w", err)

	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		IPAddresses: []net.IP{
			net.ParseIP("127.0.0.1"), // localhost
			net.ParseIP("0.0.0.0"),   // any
			net.IPv4(192, 168, 0, 0), // 192.168.0.0/16
			net.IPv4(10, 0, 0, 0),    // 10.0.0.0/8
			net.IPv4(172, 16, 0, 0),  // 172.16.0.0/12
		},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("509.CreateCertificate: %w", err)

	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("x509.MarshalPKCS8PrivateKey: %w", err)

	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("tls.X509KeyPair: %w", err)

	}

	return &tls.Config{
		Certificates:       []tls.Certificate{tlsCert},
		NextProtos:         []string{"belphegor"},
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS13,
	}, nil
}

func generateQuicConfig(keepAlive time.Duration) *quic.Config {
	return &quic.Config{
		KeepAlivePeriod: keepAlive,
	}
}
