package node

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/labi-le/belphegor/internal/channel"
	"github.com/labi-le/belphegor/internal/peer"
	"github.com/labi-le/belphegor/internal/protocol"
	"github.com/labi-le/belphegor/internal/security"
	"github.com/labi-le/belphegor/internal/transport"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/ctxlog"
	"github.com/labi-le/belphegor/pkg/id"
)

var (
	ErrAlreadyConnected = errors.New("already connected")
)

type cleanup func()

type Node struct {
	clipboard eventful.Eventful
	peers     *Storage
	channel   *channel.Channel
	transport transport.Transport
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
		case errors.Is(err, security.ErrLocalSecretMissing):
			ctxLog.Warn().Msg("i have no secrets to accept connection")
			return nil
		case errors.Is(err, security.ErrPeerSecretMissing):
			ctxLog.Trace().Msg("node that connects to us has no secrets")
			return nil
		case errors.Is(err, security.ErrSecretMismatch):
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
			msg := messageFromUpdate(update)

			if msg.Duplicate(current) && !current.Zero() {
				ctxLog.Trace().Object("msg", msg).Msg("detected duplicate")
				continue
			}

			ctxLog.Trace().Object("msg", msg).Msg("new update")

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
				ctxLog.Trace().Object("msg", msg.Payload).Msg("set clipboard data")

				if _, err := n.clipboard.Write(msg.Payload.Data); err != nil {
					ctxLog.Error().Err(err).Object("msg", msg.Payload).Send()
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

func messageFromUpdate(update eventful.Update) domain.Message {
	return domain.Message{
		ID:            id.New(),
		Data:          update.Data,
		MimeType:      update.MimeType,
		ContentHash:   update.Hash,
		ContentLength: int64(len(update.Data)),
	}
}

func (n *Node) Notify(message string, v ...any) {
	n.opts.Notifier.Notify(message, v...)
}

func (n *Node) Metadata() domain.Device {
	return n.opts.Metadata
}

func (n *Node) handleAnnounce(ctx context.Context, ann domain.EventAnnounce) {
	p, ok := n.peers.Get(ann.From)
	if !ok {
		return
	}

	logger := n.opts.Logger.With().Int64("msg_id", ann.Payload.ID).Logger()
	logger.Trace().Msg("requesting message")

	if err := p.Request(ctx, ann.Payload.ID); err != nil {
		logger.Err(err).Str("peer", p.String()).Msg("failed to request")
	}
}

func (n *Node) DiscoveryPayload() []byte {
	greet := domain.NewGreet(
		domain.WithMetadata(n.Metadata()),
		domain.WithPort(uint16(n.opts.PublicPort)),
	)
	return protocol.MustEncode(greet)
}

func (n *Node) PeerDiscovered(ctx context.Context, peerIP net.IP, payload []byte) {
	ctxLog := ctxlog.Op(n.opts.Logger, "node.PeerDiscovered")

	greet, err := protocol.DecodeExpect[domain.EventHandshake](bytes.NewReader(payload))
	if err != nil {
		ctxLog.Warn().Err(err).Msg("failed to decode discovery payload")
		return
	}

	ctxLog.Trace().
		Str("peer", greet.Payload.MetaData.String()).
		Str("addr", peerIP.String()).
		Uint32("port", greet.Payload.Port).
		Msg("discovered")

	// if metadata.IsMajorDifference(n.Metadata().Version, greet.Payload.Version) {
	//     ctxLog.Trace().Str("peer", greet.Payload.MetaData.String()).Msg("skipping peer due to version mismatch")
	//     return
	// }

	addr := fmt.Sprintf("%s:%d", peerIP.String(), greet.Payload.Port)
	_ = n.ConnectTo(ctx, addr)
}
