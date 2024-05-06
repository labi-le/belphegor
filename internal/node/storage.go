package node

import (
	"github.com/labi-le/belphegor/pkg/storage"
	"net"
	"net/netip"
)

func castAddrPort(conn net.Conn) netip.AddrPort {
	return conn.RemoteAddr().(*net.TCPAddr).AddrPort()
}

type Storage = storage.SyncMap[UniqueID, *Peer]
