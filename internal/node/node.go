package node

import (
	"context"
	"crypto"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/internal/types/proto"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/ctxlog"
	"github.com/labi-le/belphegor/pkg/encrypter"
	"github.com/labi-le/belphegor/pkg/id"
	"github.com/labi-le/belphegor/pkg/protoutil"
	pb "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	ErrAlreadyConnected = errors.New("already connected")
)

type Node struct {
	clipboard clipboard.Eventful
	peers     *Storage
	channel   *Channel
	options   Options
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

func (n *Node) addPeer(hisHand domain.Handshake, cipher *encrypter.Cipher, conn net.Conn) (*Peer, error) {
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

	peer := NewPeer(
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
func (n *Node) Start(ctx context.Context) error {
	defer n.channel.Close()

	ctxLog := ctxlog.Op("node.Start")

	var lc net.ListenConfig
	l, err := lc.Listen(ctx, "tcp4", fmt.Sprintf(":%d", n.options.PublicPort))
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
		Str("metadata", n.options.Metadata.String()).
		Msg("started")

	go n.MonitorBuffer(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil

		default:
			conn, netErr := l.Accept()
			if netErr != nil {
				if errors.Is(netErr, net.ErrClosed) {
					break
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

	hs, cipherErr := newHandshake(n.options.BitSize, n.Metadata(), n.options.PublicPort)
	if cipherErr != nil {
		return cipherErr
	}

	hisHand, cipher, greetErr := hs.exchange(conn)
	if greetErr != nil {
		return greetErr
	}

	peer, addErr := n.addPeer(
		hisHand.Payload,
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
func (n *Node) Broadcast(msg domain.EventMessage) {
	ctxLog := ctxlog.Op("node.Broadcast").
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

		// Set write timeout if the writer implements net.Conn
		err := peer.Conn().SetWriteDeadline(time.Now().Add(n.options.WriteTimeout))
		if err != nil {
			ctx.Trace().
				AnErr("SetWriteDeadline", err).
				Msg("cannot set write deadline")
			return true
		}
		defer peer.Conn().SetWriteDeadline(time.Time{})

		_, encErr := WriteEncryptedMessage(msg, peer.Signer(), peer.Conn())
		if encErr != nil {
			if errors.Is(encErr, net.ErrClosed) ||
				strings.Contains(encErr.Error(), "bad file descriptor") ||
				strings.Contains(encErr.Error(), "use of closed network connection") {

				ctx.Trace().Msg("connection closed during broadcast, removing peer")
			} else {
				ctx.Trace().
					AnErr("WriteEncrypted", encErr).
					Msg("failed to write encrypted message")
			}

			n.peers.Delete(peer.MetaData().UniqueID())
		}

		return true
	})
}

// MonitorBuffer starts monitoring the clipboard and subsequently sending data to other nodes
func (n *Node) MonitorBuffer(ctx context.Context) error {
	ctxLog := ctxlog.Op("node.MonitorBuffer")

	updates, watchErr := make(chan clipboard.Update), make(chan error, 1)
	go func() {
		if err := n.clipboard.Watch(ctx, updates); err != nil {
			watchErr <- err
		}
	}()

	go func() {
		var (
			current = domain.NewMessage((<-updates).Data)
		)
		for update := range updates {
			msg := domain.NewMessage(update.Data)
			if !msg.Duplicate(current) {
				ctxLog.
					Trace().
					Int64("msg_id", msg.ID).
					Msg("local clipboard changed")

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
			if msg.From != n.options.Metadata.UniqueID() {
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
	n.options.Notifier.Notify(message, v...)
}

func (n *Node) Metadata() domain.Device {
	return n.options.Metadata
}

func ReceiveMessage(conn net.Conn, decrypter crypto.Decrypter, data domain.Device) (domain.EventMessage, error) {
	var event proto.Event
	if decodeEnc := protoutil.DecodeReader(conn, &event); decodeEnc != nil {
		return domain.EventMessage{}, decodeEnc
	}

	payload, ok := event.Payload.(*proto.Event_Message)
	if ok == false {
		return domain.EventMessage{}, fmt.Errorf("expected: %T, actual: %T", proto.Event_Message{}, event.Payload)
	}

	return domain.MessageFromEncrypted(&event, data, func(encrypted []byte) ([]byte, error) {
		return decrypter.Decrypt(rand.Reader, payload.Message.Content, nil)
	})

}

func WriteEncryptedMessage(msg domain.EventMessage, signer crypto.Signer, writer io.Writer) (int, error) {
	dat, _ := pb.Marshal(msg.Payload.Proto())
	encrypted, err := signer.Sign(rand.Reader, dat, nil)
	if err != nil {
		return 0, err
	}

	encEv := &proto.Event{
		Created: timestamppb.New(msg.Created),
		Payload: &proto.Event_Message{Message: &proto.EncryptedMessage{ID: msg.Payload.ID, Content: encrypted}},
	}
	return protoutil.EncodeWriter(encEv, writer)
}
