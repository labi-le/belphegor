package belphegor

import (
	"net"
	"strconv"
	"sync"
)

type Storage interface {
	Add(conn net.Conn)
	Delete(hash IP)
	Get(addr IP) (net.Conn, bool)
	Exist(hash IP) bool
	All(exclude ...IP) Nodes
}

type Nodes []NodeInfo

type SyncMapStorage struct {
	m sync.Map
}

func (s *SyncMapStorage) Add(conn net.Conn) {
	host, port, _ := net.SplitHostPort(conn.RemoteAddr().String())

	portInt, _ := strconv.Atoi(port)
	s.m.Store(IP(host), NodeInfo{
		Port: portInt,
		Conn: conn,
	})
}

func (s *SyncMapStorage) Delete(hash IP) {
	s.m.Delete(hash)
}

func (s *SyncMapStorage) Get(addr IP) (net.Conn, bool) {
	v, ok := s.m.Load(addr)
	if !ok {
		return nil, false
	}
	return v.(NodeInfo).Conn, true
}

func (s *SyncMapStorage) Exist(hash IP) bool {
	_, ok := s.m.Load(hash)
	return ok
}

func (s *SyncMapStorage) All(exclude ...IP) Nodes {
	var nodes Nodes
	s.m.Range(func(key, value any) bool {
		for _, ip := range exclude {
			if ip == key.(IP) {
				return true
			}
		}

		nodes = append(nodes, NodeInfo{
			IP:   key.(IP),
			Port: value.(NodeInfo).Port,
			Conn: value.(NodeInfo).Conn,
		})
		return true
	})
	return nodes
}

func NewSyncMapStorage() Storage {
	return &SyncMapStorage{}
}
