package belphegor

import (
	"net"
	"net/netip"
	"sync"
)

// Storage represents a storage for storing nodes.
type Storage interface {
	// Add adds the specified node to the storage.
	// If the node already exists, it will be overwritten.
	Add(key netip.AddrPort, val net.Conn)
	// Delete deletes the node associated with the specified AddrPort.
	Delete(key netip.AddrPort)
	// Get netip.AddrPort the node associated with the specified AddrPort.
	Get(key netip.AddrPort) (net.Conn, bool)
	// Exist returns true if the specified node exists in the storage.
	Exist(key netip.AddrPort) bool
	// Tap calls the specified function for each node in the storage.
	Tap(fn func(netip.AddrPort, net.Conn))
}

func castAddrPortFromConn(conn net.Conn) netip.AddrPort {
	return conn.RemoteAddr().(*net.TCPAddr).AddrPort()
}

type SyncMapStorage struct {
	m sync.Map
}

// NewSyncMapStorage creates a new SyncMapStorage.
func NewSyncMapStorage() *SyncMapStorage {
	return &SyncMapStorage{}
}

func (s *SyncMapStorage) Add(key netip.AddrPort, val net.Conn) {
	s.m.Store(key, val)
}

func (s *SyncMapStorage) Delete(key netip.AddrPort) {
	s.m.Delete(key)
}

func (s *SyncMapStorage) Get(key netip.AddrPort) (net.Conn, bool) {
	v, ok := s.m.Load(key)
	if !ok {
		return nil, false
	}
	return v.(net.Conn), true
}

func (s *SyncMapStorage) Exist(key netip.AddrPort) bool {
	_, ok := s.m.Load(key)
	return ok
}

func (s *SyncMapStorage) Tap(fn func(netip.AddrPort, net.Conn)) {
	s.m.Range(func(k, v any) bool {
		fn(k.(netip.AddrPort), v.(net.Conn))
		return true
	})
}
