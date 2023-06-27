package belphegor

import (
	"belphegor/pkg/clipboard"
	"belphegor/pkg/encryption"
	"belphegor/pkg/ip"
	"github.com/rs/zerolog/log"
	"net"
)

type Node struct {
	clipboard   clipboard.Manager
	enc         *encryption.Cipher
	addr        string
	nodes       map[string]net.Conn
	lastMessage Message
}

func NewNode(clipboard clipboard.Manager, enc *encryption.Cipher, addr string) *Node {
	return &Node{
		clipboard: clipboard,
		enc:       enc,
		addr:      addr,
		nodes:     make(map[string]net.Conn),
	}
}

func NewNodeRandomPort(clipboard clipboard.Manager, enc *encryption.Cipher) *Node {
	return NewNode(clipboard, enc, ip.GetOutboundIP()+":0")
}

func (n *Node) ConnectTo(addr string) error {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	log.Info().Msgf("Connected to the clipboard: %s", addr)

	n.nodes[addr] = c
	go n.handleConnection(c)
	return nil
}

func (n *Node) Start() error {
	l, err := net.Listen("tcp", n.addr)
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

		n.nodes[conn.RemoteAddr().String()] = conn
		go n.handleConnection(conn)
	}
}

func (n *Node) handleConnection(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	externalUpdateChan := make(chan []byte)

	defer close(externalUpdateChan)
	go monitorClipboard(n, n.clipboard, 1, externalUpdateChan)
	handleClipboardData(n, conn, n.clipboard, externalUpdateChan)
}

func (n *Node) Broadcast(msg Message) {
	if n.enc != nil {
		var err error
		msg.Data, err = n.enc.Encrypt(msg.Data)
		if err != nil {
			log.Error().Msgf("failed to encrypt clipboard data: %s", err)
			return
		}
	}
	for addr, conn := range n.nodes {
		if msg.IsDuplicate(n.lastMessage) {
			continue
		}
		log.Debug().Msgf("sent message id: %s to %s: ", msg.Header.ID, addr)
		msg.Write(conn)
	}
}

func (n *Node) Close(conn net.Conn) {
	_ = conn.Close()
	delete(n.nodes, conn.RemoteAddr().String())
}
