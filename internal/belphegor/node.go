package belphegor

import (
	"belphegor/pkg/clipboard"
	"belphegor/pkg/ip"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/schollz/peerdiscovery"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"
)

const DefaultDiscoverDelay = 60 * time.Second

// IP e.g. 192.168.0.45
type IP string

type NodeInfo struct {
	net.Conn
	IP   IP
	Port int
}

type Node struct {
	clipboard      clipboard.Manager
	storage        Storage
	localClipboard Channel
	lastMessage    lastMessage
	publicPort     int
	discoverDelay  time.Duration
}

type lastMessage struct {
	Message
	mu sync.Mutex
}

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

func genPort() int {
	cryptoRand := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec dn
	return cryptoRand.Intn(1000) + 7000
}

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

func (n *Node) Start(scanDelay time.Duration) error {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", n.publicPort))
	if err != nil {
		return err
	}

	log.Info().Msgf("listening on %s", l.Addr().String())

	defer l.Close()

	go NewClipboardMonitor(n, n.clipboard, scanDelay, n.localClipboard).Start()

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
	NewNodeDataReceiver(n, conn, n.clipboard, localClipboard).Start()
}

func (n *Node) Broadcast(msg *Message, ignore ...IP) {
	defer msg.Free()

	if msg.IsDuplicate(n.GetLastMessage()) {
		return
	}

	for _, conn := range n.storage.All(ignore...) {
		log.Debug().Msgf("sent message to %s, by hash %x", conn.IP, shortHash(msg.Data.Hash))
		_, _ = msg.Write(conn)
	}
}

func (n *Node) EnableNodeDiscover() {
	_, err := peerdiscovery.NewPeerDiscovery(
		peerdiscovery.Settings{
			Payload:   []byte(strconv.Itoa(n.publicPort)),
			Limit:     -1,
			TimeLimit: -1,
			Delay:     n.discoverDelay,
			AllowSelf: false,

			Notify: func(d peerdiscovery.Discovered) {
				nodeAddr := IP(d.Address)

				if n.storage.Exist(nodeAddr) {
					log.Trace().Msgf("node %s already exist, skipping...", nodeAddr)
					return
				}
				log.Trace().Msgf("found node: %s", nodeAddr)

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

func (n *Node) SetLastMessage(msg Message) {
	n.lastMessage.mu.Lock()
	defer n.lastMessage.mu.Unlock()

	n.lastMessage.Message = msg
}

func (n *Node) GetLastMessage() Message {
	n.lastMessage.mu.Lock()
	defer n.lastMessage.mu.Unlock()

	return n.lastMessage.Message
}

func stats(storage Storage) {
	for range time.Tick(5 * time.Second) {
		nodes := storage.All()
		log.Trace().Msgf("nodes count: %d", len(nodes))
		for _, info := range nodes {
			log.Trace().Msgf("node %s %d", info.IP, info.Port)
		}

	}
}
