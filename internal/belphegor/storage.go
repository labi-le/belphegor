package belphegor

import (
	"net"
	"net/netip"
	"sync"
)

// Storage represents a storage for storing nodes.
type Storage[key any, val any] interface {
	// Add adds the specified node to the storage.
	// If the node already exists, it will be overwritten.
	Add(key key, val val)
	// Delete deletes the node associated with the specified id.
	Delete(key key)
	// Get returns the node associated with the specified id.
	Get(key key) (val, bool)
	// Exist returns true if the specified node exists in the storage.
	Exist(key key) bool
	// Tap calls the specified function for each node in the storage.
	Tap(fn func(key, val))
}

type NodeStorage Storage[UniqueID, *Peer]

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

func (s *SyncMapStorage) Add(key UniqueID, val *Peer) {
	s.m.Store(key, val)
}

func (s *SyncMapStorage) Delete(key UniqueID) {
	val, exist := s.Get(key)
	if !exist {
		return
	}
	defer val.Release()
	s.m.Delete(key)
}

func (s *SyncMapStorage) Get(key UniqueID) (*Peer, bool) {
	v, ok := s.m.Load(key)
	if !ok {
		return &Peer{}, false
	}
	return v.(*Peer), true
}

func (s *SyncMapStorage) Exist(key UniqueID) bool {
	_, ok := s.m.Load(key)
	return ok
}

func (s *SyncMapStorage) Tap(fn func(UniqueID, *Peer)) {
	s.m.Range(func(k, v any) bool {
		fn(k.(UniqueID), v.(*Peer))
		return true
	})
}
