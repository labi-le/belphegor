package netstack

import (
	"crypto/rand"
	"encoding/binary"
)

// RandomPort generates a random port number between 7000 and 7999.
func RandomPort() int {
	var b [8]byte
	_, _ = rand.Read(b[:])

	seed := binary.BigEndian.Uint64(b[:])
	return int(seed%1000) + 7000
}
