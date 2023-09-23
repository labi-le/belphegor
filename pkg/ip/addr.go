package ip

import (
	"net"
)

func MakeAddr(ip net.IP, port string) string {
	return ip.String() + MakePort(port)
}

func MakePort(port string) string {
	return ":" + port
}
