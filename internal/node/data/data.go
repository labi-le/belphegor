package data

import (
	"bytes"
	"errors"
	"github.com/labi-le/belphegor/internal/types"
)

type Data struct {
	Raw  []byte
	hash []byte
}

func DataFromProto(p *types.Data) (Data, error) {
	if p == nil {
		return Data{}, errors.New("proto data is nil")
	}

	return Data{
		Raw:  p.Raw,
		hash: p.Hash,
	}, nil
}

func (d Data) Hash() []byte {
	if d.hash == nil {
		d.hash = hashBytes(d.Raw)
	}
	return d.hash
}

func (d Data) Equal(other Data) bool {
	if len(d.Raw) == 0 || len(other.Raw) == 0 {
		return false
	}
	return bytes.Equal(d.Hash(), other.Hash())
}

func (d Data) ToProto() *types.Data {
	return &types.Data{
		Raw:  d.Raw,
		Hash: d.Hash(),
	}
}
