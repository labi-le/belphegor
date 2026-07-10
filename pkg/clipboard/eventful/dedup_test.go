package eventful_test

import (
	"bytes"
	"testing"

	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
)

var hashSizes = []struct {
	name string
	size int
}{
	{"64B", 64},
	{"1KB", 1 << 10},
	{"64KB", 64 << 10},
	{"1MB", 1 << 20},
}

// BenchmarkDeduplicator_Hash measures raw xxHash cost across payload sizes; this
// runs on every clipboard change to build the dedup key.
func BenchmarkDeduplicator_Hash(b *testing.B) {
	var d eventful.Deduplicator
	for _, sz := range hashSizes {
		data := bytes.Repeat([]byte{0xAB}, sz.size)
		b.Run(sz.name, func(b *testing.B) {
			b.SetBytes(int64(sz.size))
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				_ = d.Hash(data)
			}
		})
	}
}

// BenchmarkDeduplicator_Check measures the steady-state loop-protection path:
// the same content is re-read, hashed, and rejected as a duplicate.
func BenchmarkDeduplicator_Check(b *testing.B) {
	for _, sz := range hashSizes {
		data := bytes.Repeat([]byte{0xAB}, sz.size)
		b.Run(sz.name, func(b *testing.B) {
			var d eventful.Deduplicator
			d.Mark(data) // prime lastHash so every Check is a duplicate hit

			b.SetBytes(int64(sz.size))
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				_, _ = d.Check(data)
			}
		})
	}
}
