package network_test

import (
	"net"
	"testing"

	"github.com/labi-le/belphegor/pkg/network"
)

func TestIsLocalIP(t *testing.T) {
	tests := []struct {
		name  string
		ip    string
		local bool
	}{
		{"loopback v4", "127.0.0.1", true},
		{"loopback v6", "::1", true},
		{"unspecified v4", "0.0.0.0", true},
		{"public documentation ip", "203.0.113.1", false}, // RFC 5737 TEST-NET-3, never assigned
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := network.IsLocalIP(net.ParseIP(tt.ip)); got != tt.local {
				t.Fatalf("IsLocalIP(%s) = %v, want %v", tt.ip, got, tt.local)
			}
		})
	}
}

func TestLocalIPs_ContainsBuiltins(t *testing.T) {
	local := network.LocalIPs()
	for _, want := range []string{"127.0.0.1", "::1", "0.0.0.0", "localhost"} {
		if _, ok := local[want]; !ok {
			t.Errorf("LocalIPs() missing builtin %q", want)
		}
	}
}

func TestLocalIPs_ReturnsIndependentCopy(t *testing.T) {
	first := network.LocalIPs()
	first["injected-sentinel"] = struct{}{}

	second := network.LocalIPs()
	if _, ok := second["injected-sentinel"]; ok {
		t.Fatal("LocalIPs() must return a fresh copy; caller mutation leaked into shared state")
	}
}
