package eventful_test

import (
	"strconv"
	"testing"

	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
)

// BenchmarkUpdatesFromFileInfo measures the file-batch build path: a batch
// digest over all entries plus a per-file content hash, scaled by file count.
// It is pure computation (no filesystem access).
func BenchmarkUpdatesFromFileInfo(b *testing.B) {
	for _, n := range []int{1, 10, 100} {
		files := make([]eventful.FileInfo, n)
		for i := range files {
			files[i] = eventful.FileInfo{
				Path:    "/home/user/documents/report_" + strconv.Itoa(i) + ".pdf",
				Size:    uint64(1024 * (i + 1)),
				ModTime: uint64(1_700_000_000_000_000_000 + int64(i)),
			}
		}

		b.Run(strconv.Itoa(n)+"_files", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				_, _ = eventful.UpdatesFromFileInfo(files)
			}
		})
	}
}
