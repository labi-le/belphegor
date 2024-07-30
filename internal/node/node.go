package node

import (
	"context"
	"errors"
	"fmt"
	"github.com/labi-le/belphegor/internal/discovering"
	"github.com/labi-le/belphegor/internal/node/data"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/rs/zerolog/log"
	"time"

	"github.com/quic-go/quic-go"
)

var (
	ErrAlreadyConnected = errors.New("already connected")
)

type Connector interface {
	Connect(ctx context.Context, addr string) error
}

type Node struct {
	clipboard      clipboard.Manager
	peers          *Storage
	localClipboard data.Channel

	lastMessage *data.LastMessage

	options *Options
}

// New creates a new instance of Node with the specified settings.
func New(
	clipboard clipboard.Manager,
	peers *Storage,
	localClipboard data.Channel,
	opt *Options,
) *Node {
	if opt == nil {
		opt = defaultOptions()
	}

	return &Node{
		clipboard:      clipboard,
		peers:          peers,
		localClipboard: localClipboard,
		lastMessage:    data.NewLastMessage(),
		options:        opt,
	}
}

func gracefulShutdown(ctx context.Context, peers *Storage) {
	<-ctx.Done()
	log.Warn().Str("node.gracefulShutdown", "ctx done").Msg("close connections...")
	peers.Tap(func(id data.UniqueID, peer *Peer) {
		peer.Release()
	})
}

// Connect establishes a connection to a remote clipboard at the specified address.
// The 'addr' parameter should be in the format "host:port" to specify the remote clipboard's address
// If the connection is successfully established, it returns nil; otherwise, it returns an error
func (n *Node) Connect(ctx context.Context, addr string) error {
	conn, err := quic.DialAddr(ctx, addr, generateTLSConfig(n.options.Encryption), generateQuicConfig(n.options.KeepAlive))
	if err != nil {
		log.Error().AnErr("quic.Dial", err).Msg("failed to handle connection")
		return err
	}

	connErr := n.handleConnection(conn, true)
	if connErr != nil {
		log.Error().AnErr("node.handleConnection", connErr).Msg("failed to handle connection")
	}

	return connErr
}

func (n *Node) addPeer(hisHand *data.Greet, conn quic.Connection, stream quic.Stream) (*Peer, error) {
	metadata := data.MetaDataFromKind(hisHand.Device)
	if n.peers.Exist(metadata.UniqueID()) {
		log.Trace().Msgf("%s already connected, ignoring", metadata.String())
		return nil, ErrAlreadyConnected
	}
	peer := AcquirePeer(
		conn,
		stream,
		metadata,
		n.localClipboard,
	)

	n.peers.Add(
		metadata.UniqueID(),
		peer,
	)
	return peer, nil
}

// Start starts the node by listening for incoming connections
func (n *Node) Start(ctx context.Context) error {
	const op = "node.Start"

	go gracefulShutdown(ctx, n.peers)
	go discoverIfCan(ctx, n.options.Discovering, n.Metadata(), n)

	listener, err := quic.ListenAddr(
		fmt.Sprintf(":%d", n.options.PublicPort),
		generateTLSConfig(n.options.Encryption),
		generateQuicConfig(n.options.KeepAlive),
	)
	if err != nil {
		return err
	}
	defer listener.Close()

	log.Info().Str(op, "listen").Msgf("on %s", listener.Addr().String())
	log.Info().Str(op, "metadata").Msg(n.Metadata().String())

	defer listener.Close()

	go n.MonitorBuffer()
	go n.lastMessage.ListenUpdates()

	for {
		conn, netErr := listener.Accept(ctx)
		if netErr != nil {
			return err
		}

		log.Trace().Str(op, "accept connection").Msgf("from %s", conn.RemoteAddr().String())

		go func() {
			connErr := n.handleConnection(conn, false)
			if connErr != nil {
				log.Error().AnErr("node.handleConnection", connErr).Msg("failed to handle connection")
			}
		}()
	}
}

func discoverIfCan(ctx context.Context, options DiscoverOptions, metaData *data.MetaData, connector Connector) {
	if options.Enable {
		go discovering.New(
			options.MaxPeers,
			options.SearchDelay,
			options.Port,
		).Discover(ctx, metaData, connector)
	}
}

func generateQuicConfig(keepAlive time.Duration) *quic.Config {
	return &quic.Config{
		KeepAlivePeriod: keepAlive,
	}
}

func (n *Node) handleConnection(conn quic.Connection, client bool) error {
	var (
		stream quic.Stream
		err    error
	)
	if client {
		stream, err = conn.OpenStreamSync(conn.Context())
	} else {
		stream, err = conn.AcceptStream(conn.Context())
	}
	if err != nil {
		return err
	}

	myHand := data.NewGreet(n.options.Metadata)
	defer myHand.Release()

	log.Trace().Msgf("sending greeting to %s -> %s", myHand.Device.String(), conn.RemoteAddr())
	nw, err := data.EncodeWriter(myHand, stream)
	_ = nw
	if err != nil {
		return err
	}

	incoming, decodeErr := data.NewGreetFromReader(stream)
	if decodeErr != nil {
		return decodeErr
	}

	log.Trace().Msgf("received greeting from %s -> %s", incoming.MetaData().String(), conn.RemoteAddr().String())

	if myHand.Version != incoming.Version {
		log.Warn().Msgf("version mismatch: %s != %s", myHand.Version, incoming.Version)
	}

	peer, addErr := n.addPeer(
		incoming,
		conn,
		stream,
	)
	if addErr != nil {
		if errors.Is(addErr, ErrAlreadyConnected) {
			return nil
		}
		log.Error().AnErr("node.addPeer", addErr).Send()
		return addErr
	}
	defer n.peers.Delete(peer.MetaData().UniqueID())

	log.Info().Msgf("connected to %s", peer.String())
	peer.Receive(n.lastMessage)

	return nil
}

// Broadcast sends a message to all connected nodes except those specified in the 'ignore' list.
// It first checks if the message is a duplicate of the last sent message by comparing their IDs and hashes.
// If the message is a duplicate, it is not sent.
// For each connection in the storage, it writes the message to the connection's writer.
// The method logs the sent messages and their hashes for debugging purposes.
// The 'msg' parameter is the message to be broadcast.
// The 'ignore' parameter is a variadic list of AddrPort to exclude from the broadcast.
func (n *Node) Broadcast(msg *data.Message, ignore data.UniqueID) {
	const op = "node.Broadcast"

	defer msg.Release()

	n.peers.Tap(func(id data.UniqueID, peer *Peer) {
		if id == ignore {
			log.Trace().Str(op, "exclude sending to creator node").Msg(peer.String())
			return
		}

		if n.lastMessage.Duplicate(msg) {
			return
		}

		log.Debug().Msgf(
			"sent %s to %s",
			msg.ID(),
			peer.String(),
		)

		// Set write timeout
		reset := addWriteTimeout(peer.Stream(), n.options.WriteTimeout)
		defer reset() // Reset the deadline when done

		_, encErr := msg.Write(peer.Stream())
		if encErr != nil {
			log.Error().AnErr("message.Write", encErr).Send()
			n.peers.Delete(peer.MetaData().UniqueID())
		}
	})
}

func addWriteTimeout(stream quic.Stream, timeout time.Duration) func() error {
	if err := stream.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
		log.Error().AnErr("quic.Stream.SetWriteDeadline", err).Send()
	}
	return func() error { return stream.SetWriteDeadline(time.Time{}) }
}

// MonitorBuffer starts monitoring the clipboard and subsequently sending data to other nodes
func (n *Node) MonitorBuffer() {
	const op = "node.MonitorBuffer"
	var (
		currentClipboard = n.fetchClipboardData()
	)

	go func() {
		for range time.Tick(n.options.ClipboardScanDelay) {
			newClipboard := n.fetchClipboardData()
			if !newClipboard.Duplicate(currentClipboard) {
				log.Trace().Str(op, "local clipboard data changed").Send()

				currentClipboard = newClipboard
				n.localClipboard <- currentClipboard
			}
		}
	}()
	for msg := range n.localClipboard {
		n.setClipboardData(msg)
		n.Broadcast(msg, msg.From())
	}
}

func (n *Node) fetchClipboardData() *data.Message {
	clip, _ := n.clipboard.Get()
	return data.MessageFrom(clip, n.Metadata())
}

func (n *Node) setClipboardData(m *data.Message) {
	_ = n.clipboard.Set(m.RawData())
}

func (n *Node) Metadata() *data.MetaData {
	return n.options.Metadata
}
