package ip

import (
	"io"
	"net"
	"net/http"
)

var outBoundIP net.IP

func GetOutboundIP() string {
	if outBoundIP != nil {
		return outBoundIP.String()
	}
	req, err := http.Get("https://icanhazip.com")
	if err != nil {
		return err.Error()
	}
	defer req.Body.Close()

	body, err := io.ReadAll(req.Body)
	if err != nil {
		panic(err)
	}

	// Remove \n
	body = body[:len(body)-1]

	outBoundIP = net.ParseIP(string(body))

	return outBoundIP.String()
}

func MakeAddr(ip net.IP, port string) string {
	return ip.String() + MakePort(port)
}

func MakePort(port string) string {
	return ":" + port
}

func RemovePort(addr string) string {
	return addr[:len(addr)-6]
}

func SetOutboundIP(ip string) {
	outBoundIP = net.ParseIP(ip)
}
