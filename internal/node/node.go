package node

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"

	"github.com/labi-le/belphegor/internal/channel"
	"github.com/labi-le/belphegor/internal/peer"
	"github.com/labi-le/belphegor/internal/protocol"
	"github.com/labi-le/belphegor/internal/transport"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/ctxlog"
	"github.com/labi-le/belphegor/pkg/id"
)

var (
	ErrAlreadyConnected = errors.New("already connected")

	ErrLocalSecretMissing = errors.New("local node has no secret configured")
	ErrPeerSecretMissing  = errors.New("peer has no secret configured")
	ErrSecretMismatch     = errors.New("different secrets configured")
)

type cleanup func()

type Node struct {
	clipboard eventful.Eventful
	peers     *Storage
	channel   *channel.Channel
	transport transport.Transport // Внедренная зависимость
	opts      Options
}

func (n *Node) Close() error {
	ctxLog := ctxlog.Op(n.opts.Logger, "node.Close")

	n.peers.Tap(func(_ id.Unique, p *peer.Peer) bool {
		if closeErr := p.Close(); closeErr != nil {
			ctxLog.Warn().Err(closeErr).Str("peer", p.String()).Msg("failed to close peer")
		}
		return true
	})
	if closeErr := n.channel.Close(); closeErr != nil {
		ctxLog.Error().Err(closeErr).Msg("failed to close channel")
		return closeErr
	}

	return nil
}

// New creates a new instance of Node with the specified settings
func New(
	tr transport.Transport,
	clipboard eventful.Eventful,
	peers *Storage,
	channel *channel.Channel,
	opts ...Option,
) *Node {
	options := NewOptions(opts...)

	return &Node{
		transport: tr,
		clipboard: clipboard,
		peers:     peers,
		channel:   channel,
		opts:      options,
	}
}

// ConnectTo establishes a connection to a remote node
func (n *Node) ConnectTo(ctx context.Context, addr string) error {
	ctxLog := ctxlog.Op(n.opts.Logger, "node.ConnectTo").
		With().
		Str("addr", addr).
		Logger()

	conn, err := n.transport.Dial(ctx, addr)
	if err != nil {
		switch {
		case errors.Is(err, ErrLocalSecretMissing):
			ctxLog.Warn().Msg("i have no secrets to accept connection")
			return nil
		case errors.Is(err, ErrPeerSecretMissing):
			ctxLog.Trace().Msg("node that connects to us has no secrets")
			return nil
		case errors.Is(err, ErrSecretMismatch):
			ctxLog.Warn().Msg("we have different secrets")
			return nil
		}
		return err
	}

	if connErr := n.handleConnection(ctx, conn, false); connErr != nil {
		ctxLog.Error().AnErr("node.handleConnection", connErr).Msg("failed to handle connection")
		return connErr
	}

	return nil
}

func (n *Node) addPeer(hisHand domain.Handshake, conn transport.Connection) (*peer.Peer, cleanup, error) {
	ctxLog := ctxlog.Op(n.opts.Logger, "node.addPeer")

	metadata := hisHand.MetaData
	if n.peers.Exist(metadata.UniqueID()) {
		ctxLog.Trace().Msgf("%s already connected, ignoring", metadata.String())
		return nil, nil, ErrAlreadyConnected
	}

	pr := peer.New(
		conn,
		metadata,
		n.channel,
		n.opts.Logger,
		n.opts.Deadline,
	)

	n.peers.Add(
		metadata.UniqueID(),
		pr,
	)

	cleanup := func() {
		n.peers.Delete(metadata.UniqueID())
		n.Notify("Node disconnected %s", metadata.Name)
		_ = pr.Close()
	}

	return pr, cleanup, nil
}

// Start starts the node by listening for incoming connections
func (n *Node) Start(ctx context.Context) error {
	defer func(n *Node) {
		_ = n.Close()
		n.Notify("Bye")
	}(n)

	ctxLog := ctxlog.Op(n.opts.Logger, "node.Start")

	l, err := n.transport.Listen(ctx, fmt.Sprintf(":%d", n.opts.PublicPort))
	if err != nil {
		ctxLog.Err(err).Msg("failed to listen")
		return fmt.Errorf("node.Start: %w", err)
	}

	addr := l.Addr().String()
	n.Notify("started on %s", addr)
	ctxLog.Info().
		Str("addr", addr).
		Int64("my_node_id", n.opts.Metadata.ID).
		Type("provider", n.clipboard).
		Msg("started")

	go func() {
		if err := n.monitor(ctx); err != nil {
			ctxLog.Error().Err(err).Msg("monitor")
		}

		ctxLog.Trace().Msg("exit monitor")
	}()

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

func (n *Node) handleConnection(ctx context.Context, conn transport.Connection, accept bool) error {
	ctxLog := ctxlog.Op(n.opts.Logger, "node.handleConnection").
		With().
		Str("node", n.Metadata().String()).
		Logger()

	hs := newHandshake(n.Metadata(), n.opts.PublicPort, n.opts.Logger)
	hisHand, greetErr := hs.exchange(ctx, conn, accept)
	if greetErr != nil {
		if errors.Is(greetErr, ErrVersionMismatch) {
			return nil
		}

		return greetErr
	}

	pr, cleanup, addErr := n.addPeer(hisHand.Payload, conn)
	if addErr != nil {
		if errors.Is(addErr, ErrAlreadyConnected) {
			return nil
		}
		ctxLog.
			Err(addErr).
			Msg("failed to add")
		return addErr
	}
	defer cleanup()

	n.Notify("connected to %s", pr.MetaData().Name)

	ctxLog.Info().Msg("connected")

	return pr.Receive(ctx)
}

func openOrAcceptStream(ctx context.Context, conn transport.Connection, accept bool) (transport.Stream, error) {
	if accept {
		return conn.AcceptStream(ctx)
	}

	return conn.OpenStream(ctx)
}

func (n *Node) Broadcast(ctx context.Context, announce domain.EventAnnounce) {
	ctxLog := ctxlog.Op(n.opts.Logger, "node.Broadcast")

	blob := protocol.MustEncode(announce)
	n.peers.Tap(func(id id.Unique, peer *peer.Peer) bool {
		ctxLog := ctxLog.
			With().
			Int64("node_id", peer.MetaData().ID).
			Logger()

		if id == announce.From {
			return true
		}

		ctxLog.Trace().Msg("announced")

		encodeErr := peer.WriteContext(ctx, blob, nil)
		if encodeErr != nil {
			if errors.Is(encodeErr, net.ErrClosed) ||
				strings.Contains(encodeErr.Error(), "bad file descriptor") ||
				strings.Contains(encodeErr.Error(), "use of closed network connection") {

				ctxLog.Trace().Msg("connection closed during broadcast, removing peer")
			} else {
				ctxLog.Trace().
					AnErr("peer.Write", encodeErr).
					Msg("failed to write message")
			}

			n.peers.Delete(peer.MetaData().UniqueID())
		}

		return true
	})
}

func (n *Node) monitor(ctx context.Context) error {
	ctxLog := ctxlog.Op(n.opts.Logger, "node.monitor").With().Logger()

	updates, watchErr := make(chan eventful.Update), make(chan error, 1)
	go func() {
		defer close(watchErr)

		if err := n.clipboard.Watch(ctx, updates); err != nil {
			watchErr <- err
		}
	}()

	go func() {
		var (
			current domain.Message
		)
		for update := range updates {
			msg := domain.MessageFromUpdate(update)
			msgLog := domain.MsgLogger(ctxLog, msg.ID)

			if msg.Duplicate(current) && !current.Zero() {
				msgLog.Trace().Msg("detected duplicate")
				continue
			}

			msgLog.Trace().Msg("new update")

			current = msg
			n.channel.Send(msg.Event())
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-watchErr:
			if err != nil {
				return fmt.Errorf("node.monitor: %w", err)
			}
			return nil
		case msg, ok := <-n.channel.Messages():
			if !ok {
				return nil
			}
			if msg.From != n.opts.Metadata.UniqueID() {
				msgLog := domain.MsgLogger(ctxLog, msg.Payload.ID)
				msgLog.Trace().Msg("set clipboard data")

				if _, err := n.clipboard.Write(msg.Payload.Data); err != nil {
					msgLog.Error().Err(err).Send()
				}
			}

			n.Broadcast(ctx, domain.EventAnnounce{
				From:    msg.From,
				Created: msg.Created,
				Payload: msg.Payload.Announce(),
			})

		case ann := <-n.channel.Announcements():
			n.handleAnnounce(ctx, ann)
		}
	}
}

func (n *Node) Notify(message string, v ...any) {
	n.opts.Notifier.Notify(message, v...)
}

func (n *Node) Metadata() domain.Device {
	return n.opts.Metadata
}

func MakeTLSConfig(opts Options) (*tls.Config, error) {
	//nolint:mnd,gosec //shut up
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("rand.Int: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour * 365 * 10),
		IPAddresses: []net.IP{
			net.ParseIP("127.0.0.1"), // localhost
			net.ParseIP("0.0.0.0"),   // any
			net.IPv4(192, 168, 0, 0), // 192.168.0.0/16
			net.IPv4(10, 0, 0, 0),    // 10.0.0.0/8
			net.IPv4(172, 16, 0, 0),  // 172.16.0.0/12
		},
	}

	privateKey, publicKey, err2 := genKey(opts.Secret)
	if err2 != nil {
		return nil, err2
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("x509.CreateCertificate: %w", err)
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

	conf := &tls.Config{
		Certificates:       []tls.Certificate{tlsCert},
		NextProtos:         []string{"belphegor"},
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS13,
	}

	populateKeyLog(opts.Logger, conf)

	conf.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		if len(rawCerts) == 0 {
			return ErrPeerSecretMissing
		}

		peerCert, pErr := x509.ParseCertificate(rawCerts[0])
		if pErr != nil {
			return fmt.Errorf("failed to parse peer cert: %w", pErr)
		}

		myCert, mErr := x509.ParseCertificate(certDER)
		if mErr != nil {
			return fmt.Errorf("failed to parse my cert: %w", mErr)
		}

		myPub, myOk := myCert.PublicKey.(ed25519.PublicKey)
		peerPub, peerOk := peerCert.PublicKey.(ed25519.PublicKey)

		if myOk != peerOk {
			if myOk {
				return ErrPeerSecretMissing
			}
			return ErrLocalSecretMissing
		}

		if !myOk && !peerOk {
			return nil
		}

		if !bytes.Equal(myPub, peerPub) {
			return ErrSecretMismatch
		}

		return nil
	}

	return conf, nil
}

func genKey(secret string) (crypto.PrivateKey, crypto.PublicKey, error) {
	if secret != "" {
		seed := sha256.Sum256([]byte(secret))
		pk := ed25519.NewKeyFromSeed(seed[:])
		return pk, pk.Public(), nil
	}

	ecdsaPriv, eErr := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if eErr != nil {
		return nil, nil, fmt.Errorf("ecdsa.GenerateKey: %w", eErr)
	}
	return ecdsaPriv, ecdsaPriv.Public(), nil
}

func (n *Node) handleAnnounce(ctx context.Context, ann domain.EventAnnounce) {
	p, ok := n.peers.Get(ann.From)
	if !ok {
		return
	}

	logger := domain.MsgLogger(n.opts.Logger, ann.Payload.ID)
	logger.Trace().Msg("requesting message")

	if err := p.Request(ctx, ann.Payload.ID); err != nil {
		logger.Err(err).Str("peer", p.String()).Msg("failed to request")
	}
}
