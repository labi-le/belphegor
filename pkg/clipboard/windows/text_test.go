//go:build windows

package windows

import (
	"bytes"
	"fmt"
	"runtime"
	"syscall"
	"testing"
)

var (
	payloadASCII      = []byte("Hello World")
	payloadCyrillic   = []byte("кириллица")
	payloadEmoji      = []byte("👋 🌍 🧑‍💻 🚀")
	payloadLargeMixed = bytes.Repeat([]byte("Hello World 👋 кириллица 🌍 🧑‍💻 🚀"), 1000)
)

func withOpenClipboard(fn func() error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	r, _, _ := syscall.SyscallN(openClipboard.Addr())
	if r == 0 {
		return syscall.GetLastError()
	}
	defer syscall.SyscallN(closeClipboard.Addr())

	return fn()
}

func TestFunctional(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"ASCII", payloadASCII},
		{"Cyrillic", payloadCyrillic},
		{"Emoji", payloadEmoji},
		{"LargeMixed", payloadLargeMixed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := withOpenClipboard(func() error {
				if err := writeText(tt.data); err != nil {
					return fmt.Errorf("writeText: %w", err)
				}
				got, err := readText()
				if err != nil {
					return fmt.Errorf("readText: %w", err)
				}
				if !bytes.Equal(got, tt.data) {
					return fmt.Errorf("mismatch:\ngot len: %d\nwant len: %d", len(got), len(tt.data))
				}
				return nil
			})

			if err != nil {
				t.Fatalf("Test failed: %v", err)
			}
		})
	}
}

func BenchmarkWriteText(b *testing.B) {
	b.Run("ASCII", func(b *testing.B) {
		benchmarkWrite(b, payloadASCII)
	})
	b.Run("Cyrillic", func(b *testing.B) {
		benchmarkWrite(b, payloadCyrillic)
	})
	b.Run("Emoji", func(b *testing.B) {
		benchmarkWrite(b, payloadEmoji)
	})
	b.Run("LargeMixed", func(b *testing.B) {
		benchmarkWrite(b, payloadLargeMixed)
	})
}

func benchmarkWrite(b *testing.B, data []byte) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	r, _, _ := syscall.SyscallN(openClipboard.Addr())
	if r == 0 {
		b.Skip("Clipboard busy")
		return
	}
	defer syscall.SyscallN(closeClipboard.Addr())

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := writeText(data); err != nil {
			b.Fatal(err)
		}
	}
}
