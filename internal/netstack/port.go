package netstack

import (
	"crypto/rand"
	"encoding/binary"
)

func RandomPort() int {
	var b [8]byte
	_, _ = rand.Read(b[:])

	seed := binary.BigEndian.Uint64(b[:])
	return int(seed%1000) + 7000
}
