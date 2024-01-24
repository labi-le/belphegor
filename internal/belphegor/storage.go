package belphegor

import (
	"net"
	"strconv"
	"sync"
)

// Storage represents a storage for storing nodes.
type Storage interface {
	// Add adds the specified node to the storage.
	// If the node already exists, it will be overwritten.
	Add(conn net.Conn)
	// Delete deletes the node associated with the specified Address.
	Delete(hash Address)
	// Get returns the node associated with the specified Address.
	Get(addr Address) (net.Conn, bool)
	// Exist returns true if the specified node exists in the storage.
	Exist(hash Address) bool
	// All returns copy of all nodes excluding the specified nodes.
	All(exclude ...Address) Nodes
}

// NodeInfo represents a node's information such as Address, port, and connection.
type NodeInfo struct {
	net.Conn
	IP   Address
	Port int
}

type Nodes []NodeInfo

type SyncMapStorage struct {
	m sync.Map
}

// NewSyncMapStorage creates a new SyncMapStorage.
func NewSyncMapStorage() *SyncMapStorage {
	return &SyncMapStorage{}
}

func (s *SyncMapStorage) Add(conn net.Conn) {
	host, port, _ := net.SplitHostPort(conn.RemoteAddr().String())

	portInt, _ := strconv.Atoi(port)
	s.m.Store(Address(host), NodeInfo{
		Port: portInt,
		Conn: conn,
	})
}

func (s *SyncMapStorage) Delete(hash Address) {
	s.m.Delete(hash)
}

func (s *SyncMapStorage) Get(addr Address) (net.Conn, bool) {
	v, ok := s.m.Load(addr)
	if !ok {
		return nil, false
	}
	return v.(NodeInfo).Conn, true
}

func (s *SyncMapStorage) Exist(hash Address) bool {
	_, ok := s.m.Load(hash)
	return ok
}

func (s *SyncMapStorage) All(exclude ...Address) Nodes {
	var nodes Nodes
	s.m.Range(func(key, value any) bool {
		for _, ip := range exclude {
			if ip == key.(Address) {
				return true
			}
		}

		nodes = append(nodes, NodeInfo{
			IP:   key.(Address),
			Port: value.(NodeInfo).Port,
			Conn: value.(NodeInfo).Conn,
		})
		return true
	})
	return nodes
}
