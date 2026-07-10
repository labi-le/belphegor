package mime_test

import (
	"bytes"
	"testing"

	"github.com/labi-le/belphegor/pkg/mime"
)

// BenchmarkFrom covers the byte-sniff + classify path run on every clipboard
// read to decide how a payload is treated (text/image/path/binary).
func BenchmarkFrom(b *testing.B) {
	filler := bytes.Repeat([]byte{0x00}, 1024)

	benchmarks := []struct {
		name string
		data []byte
	}{
		{"png", append([]byte{0x89, 0x50, 0x4E, 0x47}, filler...)},
		{"jpeg", append([]byte{0xFF, 0xD8}, filler...)},
		{"gif", append([]byte{0x47, 0x49, 0x46, 0x38}, filler...)},
		{"webp", append([]byte("RIFF\x00\x00\x00\x00WEBP"), filler...)},
		{"pdf", append([]byte{0x25, 0x50, 0x44, 0x46}, filler...)},
		{"text", bytes.Repeat([]byte("lorem ipsum dolor sit amet "), 40)},
		{"empty", nil},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				_ = mime.From(bm.data)
			}
		})
	}
}

// BenchmarkAsType covers the string classifier used when the OS hands us a
// mime-type label (the common path on X11/Wayland/Windows clipboard reads).
func BenchmarkAsType(b *testing.B) {
	benchmarks := []struct {
		name     string
		mimeType string
	}{
		{"exact_hit", "text/plain"},
		{"case_normalize", "TEXT/PLAIN;charset=utf-8"},
		{"unsupported", "application/octet-stream"},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				_ = mime.AsType(bm.mimeType)
			}
		})
	}
}
