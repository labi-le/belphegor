package store_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labi-le/belphegor/internal/store"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/rs/zerolog"
)

func newStore(t *testing.T) (*store.FileStore, string) {
	t.Helper()
	dir := t.TempDir()
	fs, err := store.NewFileStore(dir, zerolog.Nop())
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	return fs, dir
}

func TestFileStore_Write_HappyPath(t *testing.T) {
	fs, base := newStore(t)
	msg := domain.Message{
		ID:            domain.MessageID(1),
		Name:          "note.txt",
		ContentLength: 5,
	}

	path, err := fs.Write(strings.NewReader("hello"), msg)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !strings.HasPrefix(path, base) {
		t.Fatalf("written path %q escapes base dir %q", path, base)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("content = %q, want %q", got, "hello")
	}
}

func TestFileStore_Write_EmptyNameRejected(t *testing.T) {
	fs, _ := newStore(t)
	if _, err := fs.Write(strings.NewReader("x"), domain.Message{ID: 1, ContentLength: 1}); err == nil {
		t.Fatal("Write with empty Name must error")
	}
}

func TestFileStore_Write_PathTraversalRejected(t *testing.T) {
	fs, base := newStore(t)
	msg := domain.Message{
		ID:            domain.MessageID(1),
		Name:          "../../evil.txt",
		ContentLength: 4,
	}

	_, err := fs.Write(strings.NewReader("evil"), msg)
	if err == nil {
		t.Fatal("Write must reject path traversal in Name")
	}
	if !strings.Contains(err.Error(), "traversal") {
		t.Fatalf("error = %v, want path traversal error", err)
	}
	if _, statErr := os.Stat(filepath.Join(filepath.Dir(base), "evil.txt")); statErr == nil {
		t.Fatal("traversal write leaked a file outside the base dir")
	}
}

func TestFileStore_Write_IncompleteContentRemovesFile(t *testing.T) {
	fs, _ := newStore(t)
	msg := domain.Message{
		ID:            domain.MessageID(7),
		Name:          "short.bin",
		ContentLength: 10, // promise 10 bytes but supply 3
	}

	path, err := fs.Write(strings.NewReader("abc"), msg)
	if err == nil {
		t.Fatal("Write must error when content is shorter than ContentLength")
	}
	if path != "" {
		t.Fatalf("path = %q, want empty on failure", path)
	}
}

func TestFileStore_Write_ExistingFileReturnsErrFileExists(t *testing.T) {
	fs, _ := newStore(t)
	msg := domain.Message{
		ID:            domain.MessageID(42),
		Name:          "dup.txt",
		ContentLength: 5,
	}

	if _, err := fs.Write(strings.NewReader("hello"), msg); err != nil {
		t.Fatalf("first Write: %v", err)
	}

	_, err := fs.Write(strings.NewReader("hello"), msg)
	if !errors.Is(err, store.ErrFileExists) {
		t.Fatalf("second Write err = %v, want ErrFileExists", err)
	}
}

func TestFileStore_Write_BatchIsolatesByBatchID(t *testing.T) {
	fs, base := newStore(t)
	msg := domain.Message{
		ID:            domain.MessageID(1),
		BatchID:       domain.MessageID(999),
		Name:          "in_batch.txt",
		ContentLength: 3,
	}

	path, err := fs.Write(strings.NewReader("abc"), msg)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	wantDir := filepath.Join(base, msg.BatchID.String())
	if filepath.Dir(path) != wantDir {
		t.Fatalf("batch file dir = %q, want %q (isolated by BatchID)", filepath.Dir(path), wantDir)
	}
}
