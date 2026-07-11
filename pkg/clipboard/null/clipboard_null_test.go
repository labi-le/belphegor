//go:build null

package null_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/clipboard/null"
	"github.com/labi-le/belphegor/pkg/mime"
	"github.com/rs/zerolog"
)

func TestNull_HeadlessDriver(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in")
	out := filepath.Join(dir, "out")
	t.Setenv("BELPHEGOR_HEADLESS_IN", in)
	t.Setenv("BELPHEGOR_HEADLESS_OUT", out)

	c := null.New(zerolog.Nop(), eventful.Options{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	upd := make(chan eventful.Update, 1)
	go func() { _ = c.Watch(ctx, upd) }()

	// A change to the IN file must surface as a local clipboard copy.
	if err := os.WriteFile(in, []byte("hello-e2e"), 0o600); err != nil {
		t.Fatal(err)
	}

	select {
	case u := <-upd:
		if string(u.Data) != "hello-e2e" {
			t.Fatalf("update data = %q, want %q", u.Data, "hello-e2e")
		}
		if u.MimeType != mime.TypeText {
			t.Errorf("update mime = %v, want text", u.MimeType)
		}
		if u.Hash == 0 {
			t.Error("update hash must be set")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no update surfaced from IN file within timeout")
	}

	// An incoming peer write must be appended to the OUT file.
	if _, err := c.Write(mime.TypeText, []byte("from-peer")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "from-peer") {
		t.Fatalf("OUT file = %q, want it to contain %q", got, "from-peer")
	}
}
