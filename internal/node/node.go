package node

import (
	"context"
	"errors"
	"fmt"
	"github.com/labi-le/belphegor/internal/netstack"
	"github.com/labi-le/belphegor/internal/notification"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/ctxlog"
	"github.com/labi-le/belphegor/pkg/encrypter"
	"github.com/rs/zerolog/log"
	"net"
	"time"
)

var (
	ErrAlreadyConnected = errors.New("already connected")
)

type Node struct {
	clipboard clipboard.Eventful
	peers     *Storage
	channel   *Channel
	options   *Options
}

type Options struct {
	PublicPort         int
	BitSize            int
	KeepAlive          time.Duration
	ClipboardScanDelay time.Duration
	WriteTimeout       time.Duration
	Notifier           notification.Notifier
	Discovering        DiscoverOptions
	Metadata           domain.MetaData
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
func WithMetadata(opt domain.MetaData) Option {
	return func(o *Options) {
		o.Metadata = opt
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
	Metadata: domain.SelfMetaData(),
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
		Metadata:           defaultOptions.Metadata,
	}

	for _, opt := range opts {
		opt(options)
	}

	return options
}

// New creates a new instance of Node with the specified settings
func New(
	clipboard clipboard.Eventful,
	peers *Storage,
	channel *Channel,
	opts ...Option,
) *Node {
	options := NewOptions(opts...)
	return &Node{
		clipboard: clipboard,
		peers:     peers,
		channel:   channel,
		options:   options,
	}
}

// ConnectTo establishes a TCP connection to a remote clipboard at the specified address
func (n *Node) ConnectTo(ctx context.Context, addr string) error {
	ctxLog := ctxlog.Op("node.ConnectTo")

	var lc net.Dialer
	conn, err := lc.DialContext(ctx, "tcp4", addr)
	if err != nil {
		ctxLog.Error().AnErr("net.Dial", err).Msg("failed to handle connection")
		return err
	}

	if connErr := n.handleConnection(ctx, conn); connErr != nil {
		ctxLog.Error().AnErr("node.handleConnection", connErr).Msg("failed to handle connection")
		return connErr
	}

	return nil
}

func (n *Node) addPeer(hisHand domain.Greet, cipher *encrypter.Cipher, conn net.Conn) (*Peer, error) {
	ctxLog := ctxlog.Op("node.addPeer")

	metadata := hisHand.MetaData
	if n.peers.Exist(metadata.UniqueID()) {
		ctxLog.Trace().Msgf("%s already connected, ignoring", metadata.String())
		return nil, ErrAlreadyConnected
	}

	if tcp, ok := conn.(*net.TCPConn); ok {
		if err := tcp.SetKeepAliveConfig(net.KeepAliveConfig{
			Enable:   true,
			Idle:     n.options.KeepAlive,
			Interval: n.options.KeepAlive,
			Count:    1,
		}); err != nil {
			return nil, err
		}
	}

	peer := AcquirePeer(
		WithConn(conn),
		WithMetaData(metadata),
		WithChannel(n.channel),
		WithCipher(cipher),
	)

	n.peers.Add(
		metadata.UniqueID(),
		peer,
	)
	return peer, nil
}

// Start starts the node by listening for incoming connections on the specified public port
func (n *Node) Start(ctx context.Context) {
	defer n.channel.Close()

	ctxLog := ctxlog.Op("node.Start")

	var lc net.ListenConfig
	l, err := lc.Listen(ctx, "tcp4", fmt.Sprintf(":%d", n.options.PublicPort))
	if err != nil {
		ctxLog.Err(err).Msg("failed to listen")
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
		Str("metadata", n.options.Metadata.String()).
		Msg("started")

	go n.MonitorBuffer(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, netErr := l.Accept()
			if netErr != nil {
				if errors.Is(netErr, net.ErrClosed) {
					return
				}
				ctxLog.
					Fatal().
					Err(netErr).
					Msg("failed to accept connection")
				return
			}

			ctxLog.
				Trace().
				Msgf("accepted connection from %s", conn.RemoteAddr())

			go func() {
				if connErr := n.handleConnection(ctx, conn); connErr != nil {
					ctxLog.
						Err(connErr).
						Msg("failed to handle connection")
				}
			}()
		}
	}
}

func (n *Node) handleConnection(ctx context.Context, conn net.Conn) error {
	ctxLog := ctxlog.Op("node.handleConnection").
		With().
		Str("node", n.Metadata().String()).
		Logger()

	hs, cipherErr := newHandshake(n.options.BitSize, n.Metadata())
	if cipherErr != nil {
		ctxLog.
			Err(cipherErr).
			Msg("generate key error")
		return cipherErr
	}

	hisHand, cipher, greetErr := hs.exchange(conn)
	if greetErr != nil {
		ctxLog.Error().
			Err(greetErr).
			Str("step", "greeting").
			Msg("greeting failed")
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

// Broadcast sends a message to all connected nodes except those specified in the 'ignore' list.
func (n *Node) Broadcast(msg domain.Message, ignore domain.UniqueID) {
	ctxLog := ctxlog.Op("node.Broadcast").
		With().
		Int64("msg_id", msg.ID()).
		Logger()

	n.peers.Tap(func(id domain.UniqueID, peer *Peer) bool {
		ctx := ctxLog.
			With().
			Str("node", peer.String()).
			Logger()

		if id == ignore {
			ctx.Trace().Msg("exclude")
			return true
		}

		ctx.Trace().Msg("sent")

		// Set write timeout if the writer implements net.Conn
		err := peer.Conn().SetWriteDeadline(time.Now().Add(n.options.WriteTimeout))
		if err != nil {
			ctx.Trace().
				AnErr("SetWriteDeadline", err).
				Msg("cannot set write deadline")
			return true
		}
		defer peer.Conn().SetWriteDeadline(time.Time{}) // Reset the deadline when done

		_, encErr := msg.WriteEncrypted(peer.Signer(), peer.Conn())
		if encErr != nil {
			ctx.Trace().
				AnErr("WriteEncrypted", encErr).
				Msg("failed to write encrypted message")
			n.peers.Delete(peer.MetaData().UniqueID())
		}

		return true
	})
}

// MonitorBuffer starts monitoring the clipboard and subsequently sending data to other nodes
func (n *Node) MonitorBuffer(ctx context.Context) {
	ctxLog := ctxlog.Op("node.MonitorBuffer")

	ch := make(chan clipboard.Update)
	go n.clipboard.Watch(ctx, ch)
	go func() {
		var (
			currentClipboard = domain.MessageFrom((<-ch).Data, n.Metadata().UniqueID())
		)
		for up := range ch {
			newClipboard := domain.MessageFrom(up.Data, n.Metadata().UniqueID())
			if !newClipboard.Duplicate(currentClipboard) {
				ctxLog.
					Trace().
					Int64("msg_id", newClipboard.ID()).
					Msg("local clipboard changed")

				currentClipboard = newClipboard
				n.channel.Send(currentClipboard)
			}
		}
		//
		//for range time.Tick(n.options.ClipboardScanDelay) {
		//	newClipboard := n.fetchClipboardData()
		//	if !newClipboard.Duplicate(currentClipboard) {
		//		ctxLog.
		//			Trace().
		//			Int64("msg_id", newClipboard.ID()).
		//			Msg("local clipboard changed")
		//
		//		currentClipboard = newClipboard
		//		n.channel.Send(currentClipboard)
		//	}
		//}
	}()
	for msg := range n.channel.Listen() {
		if msg.From() != n.options.Metadata.UniqueID() {
			log.Trace().Int64("msg_id", msg.ID()).Msg("set clipboard data")

			n.clipboard.Write(msg.RawData())
			//n.setClipboardData(msg)
		}

		go n.Broadcast(msg, msg.From())
	}
}

//func (n *Node) fetchClipboardData() domain.Message {
//	clip, _ := n.clipboard.Get()
//	return domain.MessageFrom(clip, n.Metadata().UniqueID())
//}
//
//func (n *Node) setClipboardData(m domain.Message) {
//	log.Trace().Int64("msg_id", m.ID()).Msg("set clipboard data")
//	_ = n.clipboard.Set(m.RawData())
//}

func (n *Node) Notify(message string, v ...any) {
	n.options.Notifier.Notify(message, v...)
}

func (n *Node) Metadata() domain.MetaData {
	return n.options.Metadata
}
