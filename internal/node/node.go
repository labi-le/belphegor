package node

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/labi-le/belphegor/internal/netstack"
	"github.com/labi-le/belphegor/internal/node/data"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/ip"
	"github.com/rs/zerolog/log"
	"math/big"
	"time"

	"github.com/quic-go/quic-go"
)

var (
	ErrAlreadyConnected = errors.New("already connected")
)

type Node struct {
	clipboard      clipboard.Manager
	peers          *Storage
	localClipboard data.Channel

	lastMessage *data.LastMessage

	options *Options
}

type Options struct {
	PublicPort         uint16
	BitSize            uint16
	KeepAlive          time.Duration
	ClipboardScanDelay time.Duration
	WriteTimeout       time.Duration
	// represents the current device
	Metadata *data.MetaData
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

func defaultOptions() *Options {
	return &Options{
		PublicPort:         uint16(netstack.RandomPort()),
		BitSize:            2048,
		KeepAlive:          time.Minute,
		ClipboardScanDelay: 2 * time.Second,
		WriteTimeout:       5 * time.Second,
		Metadata:           data.SelfMetaData(),
	}
}

// ConnectTo establishes a TCP connection to a remote clipboard at the specified address.
// It adds the connection to the node's storage and starts handling the connection using 'handleConnection'.
// The 'addr' parameter should be in the format "host:port" to specify the remote clipboard's address.
// If the connection is successfully established, it returns nil; otherwise, it returns an error.
func (n *Node) ConnectTo(ctx context.Context, addr string) error {
	conn, err := quic.DialAddr(ctx, addr, generateTLSConfig(), generateQuicConfig(n.options.KeepAlive))
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

// Start starts the node by listening for incoming connections on the specified public port.
// It also starts a clipboard monitor to periodically scan and update the local clipboard.
// When a new connection is accepted, it invokes the 'handleConnection' method to handle the connection.
// The 'scanDelay' parameter determines the interval at which the clipboard is scanned and updated.
// The method returns an error if it fails to start listening.
func (n *Node) Start(ctx context.Context) error {
	const op = "node.Start"

	listener, err := quic.ListenAddr(fmt.Sprintf(":%d", n.options.PublicPort), generateTLSConfig(), generateQuicConfig(n.options.KeepAlive))
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

func generateTLSConfig() *tls.Config {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	ips, _ := ip.GetLocalIPs()
	template := x509.Certificate{
		SerialNumber: serialNumber,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		IPAddresses:  ips,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	// Создаем пару ключ-сертификат
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	// Настраиваем конфигурацию TLS
	return &tls.Config{
		Certificates:       []tls.Certificate{tlsCert},
		NextProtos:         []string{"quic-echo-example"},
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: true,
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

		// Set write timeout if the writer implements net.Conn
		//err := peer.Stream().SetWriteDeadline(time.Now().Add(n.options.WriteTimeout))
		//if err != nil {
		//	log.Error().AnErr("net.Conn.SetWriteDeadline", err).Send()
		//	return
		//}
		//defer peer.Stream().SetWriteDeadline(time.Time{}) // Reset the deadline when done

		_, encErr := msg.Write(peer.Stream())
		if encErr != nil {
			log.Error().AnErr("message.WriteEncrypted", encErr).Send()
			n.peers.Delete(peer.MetaData().UniqueID())
		}
	})
}

// stats periodically log information about the nodes in the storage.
// It retrieves the list of nodes from the provided storage and logs the count of nodes
// as well as information about each node, including its Address and port.
func stats(storage *Storage) {
	for range time.Tick(time.Minute) {
		storage.Tap(func(metadata data.UniqueID, peer *Peer) {
			log.Trace().Msgf("%s is alive", peer.String())
		})
	}
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
