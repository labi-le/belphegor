package ip

import (
	"io"
	"net"
	"net/http"
	"strconv"
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

func MakeAddr(port int) string {
	if port == 0 {
		return ":"
	}
	return ":" + strconv.Itoa(port)
}

func RemovePort(addr string) string {
	return addr[:len(addr)-6]
}
