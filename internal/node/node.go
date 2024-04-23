package node

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/labi-le/belphegor/internal/types"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/encrypter"
	"github.com/labi-le/belphegor/pkg/ip"
	"github.com/labi-le/belphegor/pkg/storage"
	"github.com/rs/zerolog/log"
	"github.com/schollz/peerdiscovery"
	"google.golang.org/protobuf/proto"
	"net"
	"strconv"
	"time"
)

var (
	ErrAlreadyConnected = errors.New("already connected")
)

type Node struct {
	clipboard          clipboard.Manager
	peers              *Storage
	localClipboard     Channel
	publicPort         int
	discoverDelay      time.Duration
	bitSize            int
	keepAliveDelay     time.Duration
	clipboardScanDelay time.Duration

	lastMessage *LastMessage
}

// An Options represents options for Node
// If no options are specified, default values will be used
type Options struct {
	Clipboard          clipboard.Manager
	Peers              *Storage
	Port               int
	DiscoverDelay      time.Duration
	ClipboardChannel   Channel
	BitSize            int
	KeepAliveDelay     time.Duration
	ClipboardScanDelay time.Duration
}

func (o *Options) Prepare() {
	if o.Port <= 0 || o.Port > 65535 {
		newPort := genPort()
		log.Warn().Msgf(
			"invalid port specified: %d, use random port: %d",
			o.Port,
			newPort,
		)
		o.Port = newPort
	}

	if o.DiscoverDelay == 0 {
		o.DiscoverDelay = 1 * time.Minute
	}

	if o.ClipboardChannel == nil {
		o.ClipboardChannel = make(Channel)
	}

	if o.Peers == nil {
		o.Peers = storage.NewSyncMapStorage[UniqueID, *Peer]()
	}

	if o.Clipboard == nil {
		o.Clipboard = clipboard.NewThreadSafe()
	}

	if o.BitSize == 0 {
		o.BitSize = 2048
	}

	if o.KeepAliveDelay == 0 {
		o.KeepAliveDelay = 10 * time.Second
	}

	if o.ClipboardScanDelay == 0 {
		o.ClipboardScanDelay = 2 * time.Second
	}
}

// New creates a new instance of Node with the specified settings.
func New(opts Options) *Node {
	opts.Prepare()

	// todo rewrite to unixsocket calling method
	//go stats(opts.Peers)

	return &Node{
		clipboard:          opts.Clipboard,
		publicPort:         opts.Port,
		peers:              opts.Peers,
		discoverDelay:      opts.DiscoverDelay,
		localClipboard:     opts.ClipboardChannel,
		bitSize:            opts.BitSize,
		keepAliveDelay:     opts.KeepAliveDelay,
		clipboardScanDelay: opts.ClipboardScanDelay,
		lastMessage:        NewLastMessage(),
	}
}

// genPort generates a random port number between 7000 and 7999.
func genPort() int {
	var b [8]byte
	_, _ = rand.Read(b[:])

	seed := binary.BigEndian.Uint64(b[:])
	return int(seed%1000) + 7000
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

func (n *Node) addPeer(hisHand *types.GreetMessage, cipher *encrypter.Cipher, conn net.Conn) (*Peer, error) {
	if n.peers.Exist(hisHand.Device.UniqueID) {
		log.Trace().Msgf("%s already connected, ignoring", prettyDevice(hisHand.Device))
		return nil, ErrAlreadyConnected
	}

	if aliveErr := conn.(*net.TCPConn).SetKeepAlive(true); aliveErr != nil {
		return nil, aliveErr
	}

	if err := conn.(*net.TCPConn).SetKeepAlivePeriod(n.keepAliveDelay); err != nil {
		return nil, err
	}

	peer := AcquirePeer(
		conn,
		castAddrPort(conn),
		hisHand.Device,
		n.localClipboard,
		cipher,
	)

	n.peers.Add(
		hisHand.Device.UniqueID,
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

	l, err := net.Listen("tcp4", fmt.Sprintf(":%d", n.publicPort))
	if err != nil {
		return err
	}

	log.Info().Str(op, "listen").Msgf("on %s", l.Addr().String())
	log.Info().Str(op, "metadata").Msg(prettyDevice(thisDevice))

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
	privateKey, cipherErr := rsa.GenerateKey(rand.Reader, n.bitSize)
	if cipherErr != nil {
		log.Error().AnErr("rsa.GenerateKey", cipherErr).Send()
		return cipherErr
	}

	myGreet := greetPool.Acquire()
	myGreet.PublicKey = encrypter.PublicKey2Bytes(privateKey.Public())

	hisHand, greetErr := n.greet(myGreet, conn)
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
	defer n.peers.Delete(peer.Device().GetUniqueID())

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
func (n *Node) Broadcast(msg *Message, ignore UniqueID) {
	const op = "node.Broadcast"

	defer messagePool.Release(msg)

	n.peers.Tap(func(id UniqueID, peer *Peer) {
		if id == ignore {
			log.Trace().Msgf("%s: ignoring %s", op, peer.String())
			return
		}

		if n.lastMessage.Duplicate(msg) {
			return
		}

		log.Debug().Msgf(
			"sent %s to %s",
			msg.Header.ID,
			peer.String(),
		)

		// Set write timeout if the writer implements net.Conn
		err := peer.Conn().SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err != nil {
			log.Error().AnErr("net.Conn.SetWriteDeadline", err).Send()
			return
		}
		defer peer.Conn().SetWriteDeadline(time.Time{}) // Reset the deadline when done

		encData, encErr := peer.cipher.Sign(rand.Reader, encode(msg), nil)
		if encErr != nil {
			log.Error().AnErr("peer.cipher.Sign", encErr).Send()
		}

		if _, writeErr := encodeWriter(
			&types.EncryptedMessage{Message: encData},
			peer.Conn(),
		); writeErr != nil {
			log.Error().AnErr("encodeWriter", writeErr).Send()
			n.peers.Delete(peer.Device().GetUniqueID())
		}
	})
}

// EnableNodeDiscover enables node discovery.
// It creates a new peer discovery instance with the specified settings, including payload,
// discovery limits, time limits, delay, and whether to allow self-discovery.
// When a new node is discovered, it checks if the node already exists in the storage.
// If the node is not in the storage, it connects to the discovered node.
// If the connection attempt fails, an error is logged.
// If an error occurs while creating the peer discovery instance, the program exits with a fatal error message.
func (n *Node) EnableNodeDiscover() {
	_, err := peerdiscovery.NewPeerDiscovery(
		peerdiscovery.Settings{
			PayloadFunc: func() []byte {
				greet := greetPool.Acquire()
				defer greetPool.Release(greet)

				greet.Port = uint32(n.publicPort)

				byt, _ := proto.Marshal(greet)
				return byt
			},
			Limit:     -1,
			TimeLimit: -1,
			Delay:     n.discoverDelay,
			AllowSelf: false,

			Notify: func(d peerdiscovery.Discovered) {
				peerIP := net.ParseIP(d.Address)
				// For some reason the library calls Notify ignoring AllowSelf:false
				if ip.IsLocalIP(peerIP) {
					return
				}

				greet := greetPool.Acquire()
				defer greetPool.Release(greet)

				if protoErr := proto.Unmarshal(d.Payload, greet); protoErr != nil {
					log.Error().Err(protoErr).Msg("failed to unmarshal payload")
					return
				}

				peerAddr := fmt.Sprintf(
					"%s:%s",
					peerIP.String(),
					strconv.Itoa(int(greet.Port)),
				)
				log.Trace().Msgf("found node %s -> %s, check availability", prettyDevice(greet.Device), peerAddr)
				log.Trace().Msgf("payload: %s", greet.String())
				if err := n.ConnectTo(peerAddr); err != nil {
					log.Err(err).Msgf("failed to connect to %s -> %s", prettyDevice(greet.Device), peerAddr)
				}
			},
		},
	)

	if err != nil {
		log.Fatal().Err(err).Msg("failed to discover nodes")
	}
}

func (n *Node) greet(my *types.GreetMessage, conn net.Conn) (*types.GreetMessage, error) {
	var incoming types.GreetMessage

	log.Trace().Msgf("sending greeting to %s -> %s", prettyDevice(my.Device), conn.RemoteAddr().String())
	if _, err := encodeWriter(my, conn); err != nil {
		return &incoming, err
	}

	if decodeErr := decodeReader(conn, &incoming); decodeErr != nil {
		return &incoming, decodeErr
	}
	log.Trace().Msgf("received greeting from %s -> %s", prettyDevice(incoming.Device), conn.RemoteAddr().String())

	if my.Version != incoming.Version {
		log.Warn().Msgf("version mismatch: %s != %s", my.Version, incoming.Version)
	}
	return &incoming, nil
}

// stats periodically log information about the nodes in the storage.
// It retrieves the list of nodes from the provided storage and logs the count of nodes
// as well as information about each node, including its Address and port.
func stats(storage *Storage) {
	for range time.Tick(time.Minute) {
		storage.Tap(func(metadata UniqueID, peer *Peer) {
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
		for range time.Tick(n.clipboardScanDelay) {
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
		n.Broadcast(msg, msg.Header.From)
	}
}

func (n *Node) fetchClipboardData() *Message {
	clip, _ := n.clipboard.Get()
	return MessageFrom(clip)
}

func (n *Node) setClipboardData(m *Message) {
	_ = n.clipboard.Set(m.GetData().GetRaw())
}
