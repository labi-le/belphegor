package domain

import "crypto/sha256"

type Data struct {
	Raw  []byte
	Hash []byte
}

func NewData(raw []byte) Data {
	return Data{Raw: raw, Hash: hashBytes(raw)}
}

func hashBytes(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}
