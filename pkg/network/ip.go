package network

import "net"

var localIPs = map[string]struct{}{
	"localhost": {},
	"127.0.0.1": {},
	"::1":       {},
	// if network is down
	"0.0.0.0": {},
}

// LocalIPs will check if the IP is local
func LocalIPs() map[string]struct{} {
	local := copyMap(localIPs)

	ifaces, err := net.Interfaces()
	if err != nil {
		return local
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, address := range addrs {
			ip, _, err := net.ParseCIDR(address.String())
			if err != nil {
				continue
			}

			local[ip.String()+"%"+iface.Name] = struct{}{}
			local[ip.String()] = struct{}{}
		}
	}
	return local
}

func IsLocalIP(ip net.IP) bool {
	_, ok := LocalIPs()[ip.String()]
	return ok
}

func copyMap[T comparable, V comparable](m map[T]V) map[T]V {
	cp := make(map[T]V)
	for k, v := range m {
		cp[k] = v
	}
	return cp
}
