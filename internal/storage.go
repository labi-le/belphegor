package internal

import (
	"github.com/labi-le/belphegor/pkg/storage"
	"net"
	"net/netip"
)

func castAddrPortFromConn(conn net.Conn) netip.AddrPort {
	return conn.RemoteAddr().(*net.TCPAddr).AddrPort()
}

type NodeStorage = storage.SyncMap[UniqueID, *Peer]
