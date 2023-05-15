package ip

import (
	"net"
	"strconv"
)

func GetOutboundIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func MakeAddr(port int) string {
	if port == 0 {
		return ":"
	}
	return ":" + strconv.Itoa(port)
}

func RemovePort(addr string) string {
	return addr[:len(addr)-6]
}
