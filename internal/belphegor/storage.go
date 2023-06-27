package belphegor

import (
	"net"
	"sync"
)

type Storage interface {
	Add(conn net.Conn)
	Delete(hash NodeIP)
	Get(addr NodeIP) net.Conn
	Exist(hash NodeIP) bool
	All() map[NodeIP]NodeInfo
}

type NodeStorage struct {
	nodes      map[NodeIP]NodeInfo
	nodesMutex sync.Mutex
}

func NewNodeStorage() *NodeStorage {
	return &NodeStorage{nodes: make(map[NodeIP]NodeInfo)}
}

func (n *NodeStorage) Add(conn net.Conn) {
	host, port, _ := net.SplitHostPort(conn.RemoteAddr().String())

	n.nodes[NodeIP(host)] = NodeInfo{
		Port: port,
		Conn: conn,
	}
}

func (n *NodeStorage) Delete(hash NodeIP) {
	conn, ok := n.nodes[hash]
	if !ok {
		return
	}
	_ = conn.Close()
	delete(n.nodes, hash)
}

func (n *NodeStorage) Get(addr NodeIP) net.Conn {
	return n.nodes[addr]
}
func (n *NodeStorage) Exist(hash NodeIP) bool {
	return n.nodes[hash] != (NodeInfo{})
}

func (n *NodeStorage) All() map[NodeIP]NodeInfo {
	return n.nodes
}
