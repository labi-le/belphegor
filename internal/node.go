package internal

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	gen "github.com/labi-le/belphegor/internal/types"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/rs/zerolog/log"
	"github.com/schollz/peerdiscovery"
	"google.golang.org/protobuf/proto"
	"net"
	"strconv"
	"time"
)

const (
	DefaultDiscoverDelay = 60 * time.Second
)

var (
	ErrAlreadyConnected = errors.New("already connected")
)

type Node struct {
	clipboard      clipboard.Manager
	storage        *NodeStorage
	localClipboard Channel
	publicPort     int
	discoverDelay  time.Duration
}

// NewNode creates a new instance of Node with the specified settings.
func NewNode(
	clipboard clipboard.Manager,
	port int,
	discoverDelay time.Duration,
	storage *NodeStorage,
	channel Channel,
) *Node {
	if port <= 0 || port > 65535 {
		newPort := genPort()
		log.Warn().Msgf(
			"invalid port specified: %d, use random port: %d",
			port,
			newPort,
		)
		port = newPort
	}

	if discoverDelay == 0 {
		discoverDelay = DefaultDiscoverDelay
	}

	go stats(storage)

	return &Node{
		clipboard:      clipboard,
		publicPort:     port,
		storage:        storage,
		discoverDelay:  discoverDelay,
		localClipboard: channel,
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
		return err
	}

	go n.handleConnection(conn)
	return nil
}

func (n *Node) addPeer(hisHand *gen.GreetMessage, cipher *Cipher, conn net.Conn) (*Peer, error) {
	var peer *Peer

	if n.storage.Exist(hisHand.UniqueID) {
		log.Trace().Msgf("node %s already connected, ignoring", hisHand.UniqueID)
		return peer, ErrAlreadyConnected
	}

	rightErr := HandShake(greetPool.Acquire(), hisHand)
	if rightErr == nil {
		peer = AcquirePeer(
			conn,
			castAddrPortFromConn(conn),
			hisHand.UniqueID,
			n.localClipboard,
			cipher,
		)
		n.storage.Add(
			hisHand.UniqueID,
			peer,
		)
	}
	return peer, rightErr
}

// Start starts the node by listening for incoming connections on the specified public port.
// It also starts a clipboard monitor to periodically scan and update the local clipboard.
// When a new connection is accepted, it invokes the 'handleConnection' method to handle the connection.
// The 'scanDelay' parameter determines the interval at which the clipboard is scanned and updated.
// The method returns an error if it fails to start listening.
func (n *Node) Start(scanDelay time.Duration) error {
	l, err := net.Listen("tcp4", fmt.Sprintf(":%d", n.publicPort))
	if err != nil {
		return err
	}

	log.Info().Msgf("listening on %s", l.Addr().String())

	defer l.Close()

	go NewClipboardMonitor(n, n.clipboard, scanDelay, n.localClipboard).Receive()

	for {
		conn, netErr := l.Accept()
		if netErr != nil {
			return err
		}

		log.Info().Msgf("accepted connection from %s", conn.RemoteAddr().String())
		go n.handleConnection(conn)
	}
}

func (n *Node) handleConnection(conn net.Conn) {
	cipher := NewCipher()

	myGreet := greetPool.Acquire()
	myGreet.PublicKey = cipher.PublicKeyBytes()

	greetErr := n.greet(myGreet, conn)
	if greetErr != nil {
		log.Error().Err(greetErr).Msg("failed to greet")
		return
	}
	hisHand, errCatch := n.catchHand(conn)
	if errCatch != nil {
		log.Error().Err(errCatch).Msg("failed to catch hand")
		return
	}

	cipher.public = bytes2PublicKey(hisHand.PublicKey)
	peer, addErr := n.addPeer(hisHand, cipher, conn)
	if addErr != nil {
		if errors.Is(addErr, ErrAlreadyConnected) {
			return
		}
		log.Error().Err(addErr).Msg("failed to add peer")
		return
	}

	log.Info().Msgf("connected to the clipboard: %s", peer.ID())
	peer.Receive(n.clipboard)
	n.storage.Delete(peer.ID())
}

// Broadcast sends a message to all connected nodes except those specified in the 'ignore' list.
// It first checks if the message is a duplicate of the last sent message by comparing their IDs and hashes.
// If the message is a duplicate, it is not sent.
// For each connection in the storage, it writes the message to the connection's writer.
// The method logs the sent messages and their hashes for debugging purposes.
// The 'msg' parameter is the message to be broadcast.
// The 'ignore' parameter is a variadic list of AddrPort to exclude from the broadcast.
func (n *Node) Broadcast(msg *gen.Message, ignore UniqueID) {
	defer ReleaseMessage(msg)

	n.storage.Tap(func(id UniqueID, peer *Peer) {
		if id == ignore {
			return
		}

		if MessageIsDuplicate(peer.received.Get(), msg) {
			return
		}

		log.Debug().Msgf(
			"sent %s to %s by hash %x",
			msg.Header.ID,
			peer.ID(),
			shortHash(msg.Data.Hash),
		)

		// Set write timeout if the writer implements net.Conn
		err := peer.Conn().SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err != nil {
			log.Err(err).Msg("write timeout")
			return
		}
		defer peer.Conn().SetWriteDeadline(time.Time{}) // Reset the deadline when done

		//if _, err := encodeWriter(msg, peer.Conn()); err != nil {
		//	log.Err(err).Msg("failed to write message")
		//}

		encData, encErr := peer.cipher.Encrypt(encode(msg))
		if encErr != nil {
			log.Err(encErr).Msg("failed to encrypt message")
		}

		if _, err := encodeWriter(encData, peer.Conn()); err != nil {
			log.Err(err).Msg("failed to write message")
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
				greet := greetPool.Acquire()
				defer greetPool.Release(greet)

				if protoErr := proto.Unmarshal(d.Payload, greet); protoErr != nil {
					log.Error().Err(protoErr).Msg("failed to unmarshal payload")
					return
				}

				nodeAddr := fmt.Sprintf(
					"%s:%s",
					d.Address,
					strconv.Itoa(int(greet.Port)),
				)
				log.Info().Msgf("found node %s, check availability", nodeAddr)
				log.Trace().Msgf("payload: %s", greet.String())
				if err := n.ConnectTo(nodeAddr); err != nil {
					log.Error().Msgf("failed to connect to %s", nodeAddr)
				}
			},
		},
	)

	if err != nil {
		log.Fatal().Err(err).Msg("failed to discover nodes")
	}
}

func (n *Node) catchHand(conn net.Conn) (*gen.GreetMessage, error) {
	var greet gen.GreetMessage
	if decodeErr := decodeReader(conn, &greet); decodeErr != nil {
		return nil, decodeErr
	}
	log.Trace().Msgf("received greeting from %s", conn.RemoteAddr().String())
	return &greet, nil
}

func (n *Node) greet(my *gen.GreetMessage, conn net.Conn) error {
	log.Trace().Msgf("sending greeting to %s", conn.RemoteAddr().String())
	if _, err := encodeWriter(my, conn); err != nil {
		return err
	}
	return nil
}

// stats periodically log information about the nodes in the storage.
// It retrieves the list of nodes from the provided storage and logs the count of nodes
// as well as information about each node, including its Address and port.
func stats(storage *NodeStorage) {
	for range time.Tick(time.Minute) {
		storage.Tap(func(metadata UniqueID, peer *Peer) {
			log.Trace().Msgf("node %s is alive", metadata)
		})
	}
}
