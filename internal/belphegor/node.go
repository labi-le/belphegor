package belphegor

import (
	"belphegor/pkg/clipboard"
	"belphegor/pkg/ip"
	"github.com/rs/zerolog/log"
	"github.com/schollz/peerdiscovery"
	"math/rand"
	"net"
	"strconv"
)

// NodeIP e.g. ip
type NodeIP string

type NodeInfo struct {
	Port string
	net.Conn
}

type Node struct {
	clipboard   clipboard.Manager
	addr        string
	port        string
	storage     Storage
	lastMessage Message
}

func NewNode(clipboard clipboard.Manager, addr string) *Node {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		log.Fatal().Msgf("invalid address: %s", addr)
	}

	return &Node{
		clipboard: clipboard,
		addr:      addr,
		port:      port,
		storage:   NewNodeStorage(),
	}
}

func NewNodeRandomPort(clipboard clipboard.Manager) *Node {
	return NewNode(clipboard, genPort())
}

func genPort() string {
	// range port 7000 - 8000
	return ":" + strconv.Itoa(rand.Intn(1000)+7000)
}

func (n *Node) ConnectTo(addr string) error {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	log.Info().Msgf("Connected to the clipboard: %s", addr)

	n.storage.Add(c)

	go n.handleConnection("", c)
	return nil
}

func (n *Node) Start() error {
	l, err := net.Listen("tcp", `:`+n.port)
	if err != nil {
		return err
	}

	log.Info().Msgf("Listening on %s", l.Addr().String())

	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		log.Info().Msgf("Accepted connection from %s", conn.RemoteAddr().String())

		addr := NodeIP(conn.RemoteAddr().String())
		n.storage.Add(conn)

		go n.handleConnection(addr, conn)
	}
}

func (n *Node) handleConnection(node NodeIP, conn net.Conn) {
	defer func() { n.storage.Delete(node) }()

	externalUpdateChan := make(chan []byte)

	defer close(externalUpdateChan)
	go monitorClipboard(n, n.clipboard, 1, externalUpdateChan)
	handleClipboardData(n, conn, n.clipboard, externalUpdateChan)
}

func (n *Node) Broadcast(msg Message) {
	for addr, conn := range n.storage.All() {
		if msg.IsDuplicate(n.lastMessage) {
			continue
		}
		log.Debug().Msgf("sent message id: %s to %s: ", msg.Header.ID, addr)
		msg.Write(conn)
	}
}

func (n *Node) Close(conn net.Conn) {
	_ = conn.Close()
	n.storage.Delete(NodeIP(conn.RemoteAddr().String()))
}

func (n *Node) EnableNodeDiscover() {
	discover, err := peerdiscovery.Discover(
		peerdiscovery.Settings{
			Payload: []byte(n.port),
		},
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to discover nodes")
	}

	for _, p := range discover {
		nodeAddr := NodeIP(p.Address)
		if n.storage.Exist(nodeAddr) {
			log.Debug().Msgf("node %s already exist, skipping...", nodeAddr)
			continue
		}

		addr := ip.MakeAddr(net.ParseIP(p.Address), string(p.Payload))
		log.Info().Msgf("found node: %s", addr)
		if err := n.ConnectTo(addr); err != nil {
			log.Error().Msgf("failed to connect to %s", addr)
		}
	}
}
