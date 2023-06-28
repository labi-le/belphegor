package belphegor

import (
	"belphegor/pkg/clipboard"
	"belphegor/pkg/ip"
	"github.com/rs/zerolog/log"
	"github.com/schollz/peerdiscovery"
	"math/rand"
	"net"
	"strconv"
	"time"
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
	lastMessage *Message
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

	log.Info().Msgf("connected to the clipboard: %s", addr)

	n.storage.Add(c)

	go n.handleConnection(c)
	return nil
}

func (n *Node) Start() error {
	l, err := net.Listen("tcp", `:`+n.port)
	if err != nil {
		return err
	}

	log.Info().Msgf("listening on %s", l.Addr().String())

	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		log.Info().Msgf("accepted connection from %s", conn.RemoteAddr().String())

		n.storage.Add(conn)

		go n.handleConnection(conn)
	}
}

func (n *Node) handleConnection(conn net.Conn) {
	externalUpdateChan := make(chan []byte)

	defer close(externalUpdateChan)
	go monitorClipboard(n, n.clipboard, 2, externalUpdateChan)
	handleClipboardData(n, conn, n.clipboard, externalUpdateChan)
}

func (n *Node) Broadcast(msg *Message) {
	defer messagePool.Put(msg)

	for addr, conn := range n.storage.All() {
		if msg.IsDuplicate(n.lastMessage) {
			continue
		}

		log.Debug().Msgf("sent message id: %s to %s: ", msg.Header.ID, addr)
		msg.Write(conn)
	}
}

func (n *Node) EnableNodeDiscover() {
	_, err := peerdiscovery.NewPeerDiscovery(
		peerdiscovery.Settings{
			Payload:   []byte(n.port),
			Limit:     -1,
			TimeLimit: -1,
			Delay:     time.Second * 10,
			AllowSelf: false,

			Notify: func(d peerdiscovery.Discovered) {
				nodeAddr := NodeIP(d.Address)

				log.Trace().Msgf("found node: %s", nodeAddr)
				if n.storage.Exist(nodeAddr) {
					log.Trace().Msgf("node %s already exist, skipping...", nodeAddr)
					return
				}

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
