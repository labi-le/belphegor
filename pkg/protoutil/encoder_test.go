package protoutil_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/labi-le/belphegor/pkg/protoutil"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var payloadSizes = []struct {
	name string
	size int
}{
	{"64B", 64},
	{"1KB", 1 << 10},
	{"64KB", 64 << 10},
	{"1MB", 1 << 20},
}

// BenchmarkEncodeBytes measures the allocating wire encoder (fresh buffer per
// call) across payload sizes; it is the baseline the pooled writer improves on.
func BenchmarkEncodeBytes(b *testing.B) {
	for _, sz := range payloadSizes {
		msg := wrapperspb.Bytes(bytes.Repeat([]byte{0xAB}, sz.size))
		b.Run(sz.name, func(b *testing.B) {
			b.SetBytes(int64(sz.size))
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				if _, err := protoutil.EncodeBytes(msg); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkEncodeToWriter measures the pooled-buffer encoder that backs the hot
// send path (WriteEvent); allocations should stay flat regardless of size.
func BenchmarkEncodeToWriter(b *testing.B) {
	for _, sz := range payloadSizes {
		msg := wrapperspb.Bytes(bytes.Repeat([]byte{0xAB}, sz.size))
		b.Run(sz.name, func(b *testing.B) {
			b.SetBytes(int64(sz.size))
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				if err := protoutil.EncodeToWriter(io.Discard, msg); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkDecodeReader measures the receive path: read the length prefix, pull
// a pooled buffer, and unmarshal across payload sizes.
func BenchmarkDecodeReader(b *testing.B) {
	for _, sz := range payloadSizes {
		msg := wrapperspb.Bytes(bytes.Repeat([]byte{0xAB}, sz.size))
		encoded, err := protoutil.EncodeBytes(msg)
		if err != nil {
			b.Fatal(err)
		}

		b.Run(sz.name, func(b *testing.B) {
			r := bytes.NewReader(encoded)
			var dst wrapperspb.BytesValue
			b.SetBytes(int64(sz.size))
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				r.Reset(encoded)
				if err := protoutil.DecodeReader(r, &dst); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
