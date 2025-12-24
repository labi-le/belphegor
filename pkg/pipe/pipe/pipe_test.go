package pipe_test

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/labi-le/belphegor/pkg/pipe"
	"github.com/labi-le/belphegor/pkg/pool/byteslice"
)

func generateText(size int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var builder strings.Builder
	builder.Grow(size)

	var wordList = []string{
		"the", "quick", "brown", "fox", "jumps", "over", "lazy", "dog",
		"pack", "box", "with", "five", "dozen", "liquor", "jugs",
		"how", "vexingly", "daft", "zebras", "jump", "waltz", "nymph",
		"for", "big", "quartz", "sphinx", "of", "my", "jocks", "and",
		"love", "quest", "brought", "zero", "new", "friends", "to",
		"max", "jack", "god", "quiz", "which", "vexed", "programming",
		"code", "data", "bytes", "stream", "pipe", "buffer", "system",
		"test", "random", "content", "verify", "process", "chunk", "size",
	}

	var punctuation = []string{". ", ", ", "! ", "? ", " "}

	for builder.Len() < size {
		word := wordList[r.Intn(len(wordList))]
		builder.WriteString(word)

		punct := punctuation[r.Intn(len(punctuation))]
		builder.WriteString(punct)
	}

	result := builder.String()
	if len(result) > size {
		return result[:size]
	}
	return result
}

func TestPipe(t *testing.T) {
	t.Run("BasicSizes", func(t *testing.T) {
		pipes := []tpipe{
			newtpipe("Tiny_1KB", 1<<10),     // 1024 bytes
			newtpipe("Small_4KB", 1<<12),    // 4096 bytes
			newtpipe("Small_8KB", 1<<13),    // 8192 bytes
			newtpipe("Small_16KB", 1<<14),   // 16384 bytes
			newtpipe("Small_32KB", 1<<15),   // 32768 bytes
			newtpipe("Small_64KB", 1<<16),   // 65536 bytes
			newtpipe("Medium_128KB", 1<<17), // 131072 bytes
			newtpipe("Medium_256KB", 1<<18), // 262144 bytes
			newtpipe("Medium_512KB", 1<<19), // 524288 bytes
			newtpipe("Large_1MB", 1<<20),    // 1048576 bytes
			newtpipe("Large_2MB", 1<<21),    // 2097152 bytes
			newtpipe("Large_4MB", 1<<22),    // 4194304 bytes
			newtpipe("Large_8MB", 1<<23),    // 8388608 bytes
			newtpipe("Huge_16MB", 1<<24),    // 16777216 bytes
			newtpipe("Huge_32MB", 1<<25),    // 33554432 bytes
		}
		runPipeTests(t, pipes)
	})

	t.Run("ErrorCases", func(t *testing.T) {
		t.Run("NilReader", func(t *testing.T) {
			_, err := pipe.FromPipe(0)
			if err == nil {
				t.Error("Expected error for nil reader")
			}
		})

		t.Run("ClosedPipe", func(t *testing.T) {
			p := newtpipe("Closed_1KB", 1<<10)
			if err := p.Close(); err != nil {
				t.Error("Expected no error for method Close()")
			}

			_, err := pipe.FromPipe(p.ReadFd())
			if err == nil {
				t.Error("Expected error for closed pipe")
			}
		})
	})
}

// runPipeTests executes tests for a slice of pipe configurations
func runPipeTests(t *testing.T, pipes []tpipe) {
	for _, pip := range pipes {
		t.Run(pip.name, func(t *testing.T) {
			// Channel for synchronizing write completion
			done := make(chan error, 1)

			go func() {
				_, err := pip.Write(nil)
				done <- err
			}()

			// Read data from pipe
			buf, err := pipe.FromPipe(pip.ReadFd())
			if err != nil {
				t.Fatalf("FromPipe failed with error: %v", err)
			}
			byteslice.Put(buf)

			if err := <-done; err != nil {
				t.Fatalf("Error writing to pipe: %v", err)
			}

			if len(buf) != pip.size {
				t.Errorf("Size mismatch: got %d, expected %d",
					len(buf), pip.size)
			}

			compareData(t, buf, pip.testdata, pip.name)
		})
	}
}

// newtpipe creates a new test pipe with specified name and size
func newtpipe(name string, size int) tpipe {
	tdata := byteslice.Get(size)
	text := generateText(size)
	copy(tdata, text)

	return tpipe{
		name:     name,
		size:     len(tdata),
		Reusable: pipe.MustNonBlock(),
		testdata: tdata,
	}
}

// compareData performs detailed comparison of actual vs expected data
func compareData(t *testing.T, got []byte, want []byte, context string) {
	t.Helper()

	if len(got) != len(want) {
		t.Errorf("%s: length mismatch: got %d, expected %d",
			context, len(got), len(want))
		return
	}

	// Find first mismatch
	for i := 0; i < len(got); i++ {
		if got[i] != want[i] {
			// Show context around the error
			start := max(0, i-8)
			end := min(len(got), i+8)

			t.Errorf("%s: mismatch at position %d:\n"+
				"Got:      %s\n"+
				"Expected: %s\n"+
				"Context around error (±8 bytes):\n"+
				"Got:      %s\n"+
				"Expected: %s",
				context, i,
				formatByte(got[i]), formatByte(want[i]),
				hex.Dump(got[start:end]),
				hex.Dump(want[start:end]))
			return
		}
	}
}

func formatByte(b byte) string {
	return fmt.Sprintf("0x%02X (%d) '%c'", b, b, b)
}

type tpipe struct {
	name string
	size int
	pipe.Reusable
	testdata []byte
}

func (t *tpipe) Write(_ []byte) (n int, err error) {
	const chunkSize = 4096
	written := 0
	remaining := t.testdata

	chunk := make([]byte, chunkSize)

	for len(remaining) > 0 {
		size := chunkSize
		if len(remaining) < chunkSize {
			size = len(remaining)
		}

		copy(chunk, remaining[:size])

		n, err := syscall.Write(int(t.Fd()), chunk[:size])
		if err != nil {
			if err == syscall.EAGAIN || err == syscall.EINTR {
				continue
			}
			return written, fmt.Errorf("write error: %w", err)
		}

		written += n
		remaining = remaining[n:]
	}

	return written, nil
}

// BenchmarkFromPipe2 specifically benchmarks the FromPipe function
func BenchmarkFromPipe2(b *testing.B) {
	preparedData32KB := byteslice.Get(32 * 1024)
	defer byteslice.Put(preparedData32KB)

	tp := tpipe{
		name:     "Medium_32KB",
		size:     len(preparedData32KB),
		Reusable: pipe.MustNonBlock(),
		testdata: preparedData32KB,
	}
	defer tp.Close()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		go func() {
			tp.Write(nil)
		}()

		result, err := pipe.FromPipe(tp.ReadFd())
		if err != nil {
			b.Fatal(err)
		}

		byteslice.Put(result)
	}

}
