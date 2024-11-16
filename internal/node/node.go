package node

import (
	"errors"
	"fmt"
	"github.com/labi-le/belphegor/internal/netstack"
	"github.com/labi-le/belphegor/internal/notification"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/encrypter"
	"github.com/labi-le/belphegor/pkg/protoutil"
	"github.com/rs/zerolog/log"
	"net"
	"time"
)

var (
	ErrAlreadyConnected = errors.New("already connected")
)

type Node struct {
	clipboard      clipboard.Manager
	peers          *Storage
	localClipboard Channel
	lastMessage    *LastMessage
	options        *Options
}

type Options struct {
	PublicPort         int
	BitSize            int
	KeepAlive          time.Duration
	ClipboardScanDelay time.Duration
	WriteTimeout       time.Duration
	Notifier           notification.Notifier
	Discovering        DiscoverOptions
}

type DiscoverOptions struct {
	Enable   bool
	Delay    time.Duration
	MaxPeers int
}

// Option defines the method to configure Options
type Option func(*Options)

func WithPublicPort(port int) Option {
	return func(o *Options) {
		o.PublicPort = port
	}
}

func WithBitSize(size int) Option {
	return func(o *Options) {
		o.BitSize = size
	}
}

func WithKeepAlive(duration time.Duration) Option {
	return func(o *Options) {
		o.KeepAlive = duration
	}
}

func WithClipboardScanDelay(delay time.Duration) Option {
	return func(o *Options) {
		o.ClipboardScanDelay = delay
	}
}

func WithWriteTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.WriteTimeout = timeout
	}
}

func WithNotifier(notifier notification.Notifier) Option {
	return func(o *Options) {
		o.Notifier = notifier
	}
}

func WithDiscovering(opt DiscoverOptions) Option {
	return func(o *Options) {
		o.Discovering = opt
	}
}

var defaultOptions = &Options{
	PublicPort:         netstack.RandomPort(),
	BitSize:            2048,
	KeepAlive:          time.Minute,
	ClipboardScanDelay: 2 * time.Second,
	WriteTimeout:       5 * time.Second,
	Notifier:           new(notification.BeepDecorator),
	Discovering: DiscoverOptions{
		Enable:   true,
		Delay:    5 * time.Minute,
		MaxPeers: 5,
	},
}

// NewOptions creates Options with provided options
func NewOptions(opts ...Option) *Options {
	options := &Options{
		PublicPort:         defaultOptions.PublicPort,
		BitSize:            defaultOptions.BitSize,
		KeepAlive:          defaultOptions.KeepAlive,
		ClipboardScanDelay: defaultOptions.ClipboardScanDelay,
		WriteTimeout:       defaultOptions.WriteTimeout,
		Notifier:           defaultOptions.Notifier,
		Discovering:        defaultOptions.Discovering,
	}

	for _, opt := range opts {
		opt(options)
	}

	return options
}

// New creates a new instance of Node with the specified settings
func New(
	clipboard clipboard.Manager,
	peers *Storage,
	localClipboard Channel,
	opts ...Option,
) *Node {
	return &Node{
		clipboard:      clipboard,
		peers:          peers,
		localClipboard: localClipboard,
		lastMessage:    NewLastMessage(),
		options:        NewOptions(opts...),
	}
}

// ConnectTo establishes a TCP connection to a remote clipboard at the specified address
func (n *Node) ConnectTo(addr string) error {
	ctxLog := log.With().Str("op", "node.ConnectTo").Logger()

	conn, err := net.Dial("tcp4", addr)
	if err != nil {
		ctxLog.Error().AnErr("net.Dial", err).Msg("failed to handle connection")
		return err
	}

	if connErr := n.handleConnection(conn); connErr != nil {
		ctxLog.Error().AnErr("node.handleConnection", connErr).Msg("failed to handle connection")
		return connErr
	}

	return nil
}

func (n *Node) addPeer(hisHand domain.Greet, cipher *encrypter.Cipher, conn net.Conn) (*Peer, error) {
	ctxLog := log.With().Str("op", "node.addPeer").Logger()

	metadata := hisHand.MetaData
	if n.peers.Exist(metadata.UniqueID()) {
		ctxLog.Trace().Msgf("%s already connected, ignoring", metadata.String())
		return nil, ErrAlreadyConnected
	}

	if aliveErr := conn.(*net.TCPConn).SetKeepAlive(true); aliveErr != nil {
		return nil, aliveErr
	}

	if err := conn.(*net.TCPConn).SetKeepAlivePeriod(n.options.KeepAlive); err != nil {
		return nil, err
	}

	peer := AcquirePeer(
		WithConn(conn),
		WithMetaData(metadata),
		WithLocalClipboard(n.localClipboard),
		WithCipher(cipher),
	)

	n.peers.Add(
		metadata.UniqueID(),
		peer,
	)
	return peer, nil
}

// Start starts the node by listening for incoming connections on the specified public port
func (n *Node) Start() error {
	ctxLog := log.With().Str("op", "node.Start").Logger()

	l, err := net.Listen("tcp4", fmt.Sprintf(":%d", n.options.PublicPort))
	if err != nil {
		return err
	}
	defer l.Close()

	addr := l.Addr().String()
	metadata := domain.SelfMetaData()

	n.Notify("started on %s", addr)
	ctxLog.Info().
		Str("address", addr).
		Str("metadata", metadata.String()).
		Msg("node started")

	go n.MonitorBuffer()
	go n.lastMessage.ListenUpdates()

	for {
		conn, netErr := l.Accept()
		if netErr != nil {
			return err
		}

		ctxLog.Trace().Msgf("accepted connection from %s", conn.RemoteAddr())

		go func() {
			if connErr := n.handleConnection(conn); connErr != nil {
				ctxLog.Err(connErr).Msg("failed to handle connection")
			}
		}()
	}
}

func (n *Node) handleConnection(conn net.Conn) error {
	ctxLog := log.With().Str("op", "node.handleConnection").Logger()

	hs, cipherErr := newHandshake(n.options.BitSize)
	if cipherErr != nil {
		ctxLog.Err(cipherErr).Msg("failed to generate key")
		return cipherErr
	}

	hisHand, cipher, greetErr := hs.exchange(conn)
	if greetErr != nil {
		ctxLog.Err(greetErr).Msg("failed to greet")
		return greetErr
	}

	peer, addErr := n.addPeer(
		hisHand,
		cipher,
		conn,
	)
	if addErr != nil {
		if errors.Is(addErr, ErrAlreadyConnected) {
			return nil
		}
		ctxLog.Err(addErr).Msg("failed to add peer")
		return addErr
	}
	defer n.peers.Delete(peer.MetaData().UniqueID())

	n.Notify("connected to %s", peer.MetaData().Name)
	defer n.Notify("Node disconnected %s", peer.MetaData().Name)

	ctxLog.Info().Str("peer", peer.String()).Msg("connected")

	peer.Receive(n.lastMessage)
	return nil
}

// Broadcast sends a message to all connected nodes except those specified in the 'ignore' list.
func (n *Node) Broadcast(msg *domain.Message, ignore domain.UniqueID) {
	ctxLog := log.With().Str("op", "node.Broadcast").Logger()

	n.peers.Tap(func(id domain.UniqueID, peer *Peer) {
		if id == ignore {
			ctxLog.Trace().Msgf("exclude sending to creator node: %s", peer.String())
			return
		}

		if n.lastMessage.Duplicate(msg) {
			return
		}

		ctxLog.Debug().Msgf(
			"sent %d to %s",
			msg.ID(),
			peer.String(),
		)

		// Set write timeout if the writer implements net.Conn
		err := peer.Conn().SetWriteDeadline(time.Now().Add(n.options.WriteTimeout))
		if err != nil {
			ctxLog.Error().AnErr("net.Conn.SetWriteDeadline", err).Send()
			return
		}
		defer peer.Conn().SetWriteDeadline(time.Time{}) // Reset the deadline when done

		_, encErr := msg.WriteEncrypted(peer.Signer(), peer.Conn())
		if encErr != nil {
			ctxLog.Error().AnErr("message.WriteEncrypted", encErr).Send()
			n.peers.Delete(peer.MetaData().UniqueID())
		}
	})
}

func (n *Node) greet(my domain.Greet, conn net.Conn) (domain.Greet, error) {
	if _, err := protoutil.EncodeWriter(&my, conn); err != nil {
		return domain.Greet{}, err
	}

	incoming, decodeErr := domain.NewGreetFromReader(conn)
	if decodeErr != nil {
		return incoming, decodeErr
	}

	ctxLog := log.With().Str("op", "node.greet").Logger()
	ctxLog.Trace().Msgf("received greeting from %s -> %s", incoming.MetaData.String(), conn.RemoteAddr().String())

	if my.Version != incoming.Version {
		ctxLog.Warn().Msgf("version mismatch: %s != %s", my.Version, incoming.Version)
	}
	return incoming, nil
}

// MonitorBuffer starts monitoring the clipboard and subsequently sending data to other nodes
func (n *Node) MonitorBuffer() {
	ctxLog := log.With().Str("op", "node.MonitorBuffer").Logger()

	var (
		currentClipboard = n.fetchClipboardData()
	)

	go func() {
		for range time.Tick(n.options.ClipboardScanDelay) {
			newClipboard := n.fetchClipboardData()
			if !newClipboard.Duplicate(currentClipboard) {
				ctxLog.Trace().Msg("local clipboard data changed")

				currentClipboard = newClipboard
				n.localClipboard <- currentClipboard
			}
		}
	}()
	for msg := range n.localClipboard {
		if !msg.My() {
			n.setClipboardData(msg)
		}
		n.Broadcast(msg, msg.From())
	}
}

func (n *Node) fetchClipboardData() *domain.Message {
	clip, _ := n.clipboard.Get()
	return domain.MessageFrom(clip)
}

func (n *Node) setClipboardData(m *domain.Message) {
	log.Trace().Msg("set")
	_ = n.clipboard.Set(m.RawData())
}

func (n *Node) Notify(message string, v ...any) {
	n.options.Notifier.Notify(message, v...)
}
