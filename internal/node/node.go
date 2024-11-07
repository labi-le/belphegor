package node

import (
	"crypto/rand"
	"crypto/rsa"
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

// New creates a new instance of Node with the specified settings.
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

// ConnectTo establishes a TCP connection to a remote clipboard at the specified address.
// It adds the connection to the node's storage and starts handling the connection using 'handleConnection'.
// The 'addr' parameter should be in the format "host:port" to specify the remote clipboard's address.
// If the connection is successfully established, it returns nil; otherwise, it returns an error.
func (n *Node) ConnectTo(addr string) error {
	conn, err := net.Dial("tcp4", addr)
	if err != nil {
		log.Error().AnErr("net.Dial", err).Msg("failed to handle connection")
		return err
	}

	connErr := n.handleConnection(conn)
	if connErr != nil {
		log.Error().AnErr("node.handleConnection", connErr).Msg("failed to handle connection")
	}

	return connErr
}

func (n *Node) addPeer(hisHand domain.Greet, cipher *encrypter.Cipher, conn net.Conn) (*Peer, error) {
	metadata := hisHand.MetaData
	if n.peers.Exist(metadata.UniqueID()) {
		log.Trace().Msgf("%s already connected, ignoring", metadata.String())
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

// Start starts the node by listening for incoming connections on the specified public port.
// It also starts a clipboard monitor to periodically scan and update the local clipboard.
// When a new connection is accepted, it invokes the 'handleConnection' method to handle the connection.
// The 'scanDelay' parameter determines the interval at which the clipboard is scanned and updated.
// The method returns an error if it fails to start listening.
func (n *Node) Start() error {
	const op = "node.Start"

	l, err := net.Listen("tcp4", fmt.Sprintf(":%d", n.options.PublicPort))
	if err != nil {
		return err
	}

	n.Notify("started on %s", l.Addr().String())

	log.Info().Str(op, "listen").Msgf("on %s", l.Addr().String())
	log.Info().Str(op, "metadata").Msg(domain.SelfMetaData().String())

	defer l.Close()

	go n.MonitorBuffer()
	go n.lastMessage.ListenUpdates()

	for {
		conn, netErr := l.Accept()
		if netErr != nil {
			return err
		}

		log.Trace().Str(op, "accept connection").Msgf("from %s", conn.RemoteAddr().String())
		go func() {
			connErr := n.handleConnection(conn)
			if connErr != nil {
				log.Error().AnErr("node.handleConnection", connErr).Msg("failed to handle connection")
			}
		}()
	}
}

func (n *Node) handleConnection(conn net.Conn) error {
	privateKey, cipherErr := rsa.GenerateKey(rand.Reader, int(n.options.BitSize))
	if cipherErr != nil {
		log.Error().AnErr("rsa.GenerateKey", cipherErr).Send()
		return cipherErr
	}

	hisHand, greetErr := n.greet(
		domain.NewGreet(domain.WithPublicKey(encrypter.PublicKey2Bytes(privateKey.Public()))),
		conn,
	)
	if greetErr != nil {
		log.Error().AnErr("node.greet", greetErr).Send()
		return greetErr
	}

	peer, addErr := n.addPeer(
		hisHand,
		encrypter.NewCipher(privateKey, encrypter.Bytes2PublicKey(hisHand.PublicKey)),
		conn,
	)
	if addErr != nil {
		if errors.Is(addErr, ErrAlreadyConnected) {
			return nil
		}
		log.Error().AnErr("node.addPeer", addErr).Send()
		return addErr
	}
	defer n.peers.Delete(peer.MetaData().UniqueID())
	n.Notify("connected to %s", peer.MetaData().Name)
	defer n.Notify("Node disconnected %s", peer.MetaData().Name)

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
func (n *Node) Broadcast(msg *domain.Message, ignore domain.UniqueID) {
	const op = "node.Broadcast"

	n.peers.Tap(func(id domain.UniqueID, peer *Peer) {
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

		// Set write timeout if the writer implements net.Conn
		err := peer.Conn().SetWriteDeadline(time.Now().Add(n.options.WriteTimeout))
		if err != nil {
			log.Error().AnErr("net.Conn.SetWriteDeadline", err).Send()
			return
		}
		defer peer.Conn().SetWriteDeadline(time.Time{}) // Reset the deadline when done

		_, encErr := msg.WriteEncrypted(peer.Signer(), peer.Conn())
		if encErr != nil {
			log.Error().AnErr("message.WriteEncrypted", encErr).Send()
			n.peers.Delete(peer.MetaData().UniqueID())
		}
	})
}

func (n *Node) greet(my domain.Greet, conn net.Conn) (domain.Greet, error) {
	log.Trace().Msgf("sending greeting to %s -> %s", my.MetaData.String(), conn.RemoteAddr().String())
	if _, err := protoutil.EncodeWriter(&my, conn); err != nil {
		return domain.Greet{}, err
	}

	incoming, decodeErr := domain.NewGreetFromReader(conn)
	if decodeErr != nil {
		return incoming, decodeErr
	}

	log.Trace().Msgf("received greeting from %s -> %s", incoming.MetaData.String(), conn.RemoteAddr().String())

	if my.Version != incoming.Version {
		log.Warn().Msgf("version mismatch: %s != %s", my.Version, incoming.Version)
	}
	return incoming, nil
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

func (n *Node) fetchClipboardData() *domain.Message {
	clip, _ := n.clipboard.Get()
	return domain.MessageFrom(clip)
}

func (n *Node) setClipboardData(m *domain.Message) {
	_ = n.clipboard.Set(m.RawData())
}

func (n *Node) Notify(message string, v ...any) {
	n.options.Notifier.Notify(message, v...)
}
