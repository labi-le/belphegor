package eventful_test

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/mime"
)

func TestDeduplicator_Check(t *testing.T) {
	var d eventful.Deduplicator
	data := []byte("clipboard payload")

	h1, isNew := d.Check(data)
	if !isNew {
		t.Fatal("first Check of fresh data must report new")
	}

	h2, isNew := d.Check(data)
	if isNew {
		t.Fatal("repeat Check of same data must report duplicate")
	}
	if h1 != h2 {
		t.Fatalf("hash unstable for same data: %d != %d", h1, h2)
	}

	h3, isNew := d.Check([]byte("different payload"))
	if !isNew {
		t.Fatal("Check of changed data must report new")
	}
	if h3 == h1 {
		t.Fatal("distinct payloads should not share a hash")
	}
}

func TestDeduplicator_Mark(t *testing.T) {
	var d eventful.Deduplicator
	data := []byte("payload")

	d.Mark(data) // prime lastHash without emitting

	if _, isNew := d.Check(data); isNew {
		t.Fatal("Check after Mark of same data must report duplicate")
	}
}

func TestDeduplicator_Hash(t *testing.T) {
	var d eventful.Deduplicator
	h1 := d.Hash([]byte("stable"))
	h2 := d.Hash([]byte("stable"))
	if h1 != h2 {
		t.Fatal("Hash must be deterministic")
	}
	if d.Hash([]byte("a")) == d.Hash([]byte("b")) {
		t.Fatal("distinct inputs must not collide")
	}
}

func TestUpdatesFromFileInfo(t *testing.T) {
	files := []eventful.FileInfo{
		{Path: "/tmp/a.txt", Size: 10, ModTime: 111},
		{Path: "/tmp/b.txt", Size: 20, ModTime: 222},
	}

	updates, batchHash := eventful.UpdatesFromFileInfo(files)
	if len(updates) != len(files) {
		t.Fatalf("got %d updates, want %d", len(updates), len(files))
	}
	if len(batchHash) == 0 {
		t.Fatal("batch hash must not be empty")
	}

	batchID := updates[0].BatchID
	for i, u := range updates {
		if u.MimeType != mime.TypePath {
			t.Errorf("update %d mime = %v, want path", i, u.MimeType)
		}
		if u.BatchTotal != uint32(len(files)) {
			t.Errorf("update %d batchTotal = %d, want %d", i, u.BatchTotal, len(files))
		}
		if u.BatchID != batchID {
			t.Errorf("update %d batchID = %d, want shared %d", i, u.BatchID, batchID)
		}
		if u.Size != files[i].Size {
			t.Errorf("update %d size = %d, want %d", i, u.Size, files[i].Size)
		}
		if string(u.Data) != files[i].Path {
			t.Errorf("update %d data = %q, want %q", i, u.Data, files[i].Path)
		}
	}
	if updates[0].Hash == updates[1].Hash {
		t.Error("distinct files must have distinct per-file hashes")
	}
}

func TestUpdatesFromFileInfo_Deterministic(t *testing.T) {
	files := []eventful.FileInfo{{Path: "/x", Size: 1, ModTime: 2}}

	u1, h1 := eventful.UpdatesFromFileInfo(files)
	u2, h2 := eventful.UpdatesFromFileInfo(files)

	if string(h1) != string(h2) {
		t.Fatal("batch hash not deterministic")
	}
	if u1[0].BatchID != u2[0].BatchID || u1[0].Hash != u2[0].Hash {
		t.Fatal("update identifiers not deterministic")
	}
}

func TestUpdatesFromRawPath(t *testing.T) {
	dir := t.TempDir()
	fileA := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(fileA, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	fileB := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(fileB, []byte("worldwide"), 0o644); err != nil {
		t.Fatal(err)
	}
	subdir := filepath.Join(dir, "sub")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	raw := "file://" + fileA + "\r\n" +
		"file://" + fileB + "\r\n" +
		"file://" + subdir + "\r\n" + // directory -> skipped
		"file://" + filepath.Join(dir, "missing.txt") + "\r\n" + // nonexistent -> skipped
		"plain text line\r\n" // not a file uri -> skipped

	updates, batchHash := eventful.UpdatesFromRawPath([]byte(raw), 0)
	if len(updates) != 2 {
		t.Fatalf("got %d updates, want 2 (dir/missing/non-uri skipped)", len(updates))
	}
	if len(batchHash) == 0 {
		t.Fatal("batch hash must not be empty")
	}

	sizeByPath := map[string]uint64{}
	for _, u := range updates {
		if u.MimeType != mime.TypePath {
			t.Errorf("mime = %v, want path", u.MimeType)
		}
		sizeByPath[string(u.Data)] = u.Size
	}
	if sizeByPath[fileA] != 5 {
		t.Errorf("size(%s) = %d, want 5", fileA, sizeByPath[fileA])
	}
	if sizeByPath[fileB] != 9 {
		t.Errorf("size(%s) = %d, want 9", fileB, sizeByPath[fileB])
	}
}

func TestUpdatesFromRawPath_Limit(t *testing.T) {
	dir := t.TempDir()
	var raw []byte
	for i := range 3 {
		p := filepath.Join(dir, "f"+strconv.Itoa(i)+".txt")
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		raw = append(raw, "file://"+p+"\r\n"...)
	}

	updates, _ := eventful.UpdatesFromRawPath(raw, 1)
	if len(updates) != 1 {
		t.Fatalf("got %d updates, want 1 with limit=1", len(updates))
	}
}

func TestUpdatesFromRawPath_PercentEncoded(t *testing.T) {
	dir := t.TempDir()
	decoded := filepath.Join(dir, "a b.txt") // literal space in name
	if err := os.WriteFile(decoded, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	raw := "file://" + dir + "/a%20b.txt\r\n"
	updates, _ := eventful.UpdatesFromRawPath([]byte(raw), 0)
	if len(updates) != 1 {
		t.Fatalf("got %d updates, want 1 for percent-encoded path", len(updates))
	}
	if string(updates[0].Data) != decoded {
		t.Errorf("data = %q, want decoded %q", updates[0].Data, decoded)
	}
}

func TestUpdatesFromRawPath_NoValidReturnsEmpty(t *testing.T) {
	updates, batchHash := eventful.UpdatesFromRawPath([]byte("no uris here\r\n"), 0)
	if len(updates) != 0 {
		t.Fatalf("got %d updates, want 0", len(updates))
	}
	if batchHash != nil {
		t.Errorf("batchHash = %v, want nil", batchHash)
	}
}
