package belphegor

import (
	"belphegor/pkg/clipboard"
	"belphegor/pkg/ip"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/schollz/peerdiscovery"
	"net"
	"strconv"
	"sync"
	"time"
)

const DefaultDiscoverDelay = 60 * time.Second

// Address e.g. 192.168.0.45
type Address string

type Node struct {
	clipboard      clipboard.Manager
	storage        Storage
	localClipboard Channel
	lastMessage    lastMessage
	publicPort     int
	discoverDelay  time.Duration
}

// lastMessage which is stored in Node and serves to identify duplicate messages
type lastMessage struct {
	*Message
	mu sync.Mutex
}

// NewNode creates a new instance of Node with the specified settings.
func NewNode(
	clipboard clipboard.Manager,
	port int,
	discoverDelay time.Duration,
	storage Storage,
	channel Channel,
) *Node {
	if port <= 0 {
		log.Fatal().Msgf("invalid publicPort: %d", port)
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

// NewNodeRandomPort creates a new instance of Node with a random port number.
func NewNodeRandomPort(
	clipboard clipboard.Manager,
	discoverDelay time.Duration,
	storage Storage,
	channel Channel,
) *Node {
	return NewNode(
		clipboard,
		genPort(),
		discoverDelay,
		storage,
		channel,
	)
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
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	log.Info().Msgf("connected to the clipboard: %s", addr)

	n.storage.Add(c)

	go n.handleConnection(c, n.localClipboard)
	return nil
}

// Start starts the node by listening for incoming connections on the specified public port.
// It also starts a clipboard monitor to periodically scan and update the local clipboard.
// When a new connection is accepted, it invokes the 'handleConnection' method to handle the connection.
// The 'scanDelay' parameter determines the interval at which the clipboard is scanned and updated.
// The method returns an error if it fails to start listening.
func (n *Node) Start(scanDelay time.Duration) error {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", n.publicPort))
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

		go n.handleConnection(conn, n.localClipboard)

	}
}

func (n *Node) handleConnection(conn net.Conn, localClipboard Channel) {
	n.storage.Add(conn)
	NewNodeDataReceiver(n, conn, n.clipboard, localClipboard).Receive()
}

// Broadcast sends a message to all connected nodes except those specified in the 'ignore' list.
// It first checks if the message is a duplicate of the last sent message by comparing their IDs and hashes.
// If the message is a duplicate, it is not sent.
// For each connection in the storage, it writes the message to the connection's writer.
// The method logs the sent messages and their hashes for debugging purposes.
// The 'msg' parameter is the message to be broadcasted.
// The 'ignore' parameter is a variadic list of Address addresses to exclude from the broadcast.
func (n *Node) Broadcast(msg *Message, ignore ...Address) {
	defer msg.Release()

	if MessageIsDuplicate(n.GetLastMessage(), msg) {
		return
	}

	for _, conn := range n.storage.All(ignore...) {
		log.Debug().Msgf("sent %s to %s by hash %x", msg.Header.ID, conn.IP, shortHash(msg.Data.Hash))
		_, _ = msg.Write(conn)
	}
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
			Payload:   []byte(strconv.Itoa(n.publicPort)),
			Limit:     -1,
			TimeLimit: -1,
			Delay:     n.discoverDelay,
			AllowSelf: false,

			Notify: func(d peerdiscovery.Discovered) {
				nodeAddr := Address(d.Address)

				if n.storage.Exist(nodeAddr) {
					log.Trace().Msgf("node %s already exist, skipping...", nodeAddr)
					return
				}
				log.Info().Msgf("found node: %s", nodeAddr)

				addr := ip.MakeAddr(net.ParseIP(d.Address), string(d.Payload))
				log.Info().Msgf("connecting to %s", addr)
				if err := n.ConnectTo(addr); err != nil {
					log.Error().Msgf("failed to connect to %s", addr)
				}
			},
		},
	)

	if err != nil {
		log.Fatal().Err(err).Msg("failed to discover nodes")
	}
}

func (n *Node) SetLastMessage(msg *Message) {
	n.lastMessage.mu.Lock()
	defer n.lastMessage.mu.Unlock()

	n.lastMessage.Message = msg
}

func (n *Node) GetLastMessage() *Message {
	n.lastMessage.mu.Lock()
	defer n.lastMessage.mu.Unlock()

	return n.lastMessage.Message
}

// stats periodically logs information about the nodes in the storage.
// It retrieves the list of nodes from the provided storage and logs the count of nodes
// as well as information about each node, including its Address address and port.
// The function runs at an interval of 5 seconds.
func stats(storage Storage) {
	for range time.Tick(5 * time.Second) {
		nodes := storage.All()
		log.Trace().Msgf("nodes count: %d", len(nodes))
		for _, info := range nodes {
			log.Trace().Msgf("node %s %d", info.IP, info.Port)
		}
	}
}
