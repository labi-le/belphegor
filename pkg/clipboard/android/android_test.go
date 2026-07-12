package android

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/mime"
	"github.com/rs/zerolog"
)

type capturingWriter struct {
	mu    sync.Mutex
	mimes []string
	datas [][]byte
}

func (w *capturingWriter) Write(mimeType string, data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.mimes = append(w.mimes, mimeType)
	cp := make([]byte, len(data))
	copy(cp, data)
	w.datas = append(w.datas, cp)
	return nil
}

func (w *capturingWriter) calls() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.mimes)
}

func newClip(w Writer) *Clipboard {
	return New(zerolog.Nop(), eventful.Options{}, w)
}

// drainOne reads a single Update or fails after a short timeout.
func drainOne(t *testing.T, ch <-chan eventful.Update) eventful.Update {
	t.Helper()
	select {
	case u := <-ch:
		return u
	case <-time.After(time.Second):
		t.Fatal("expected an update, got none")
		return eventful.Update{}
	}
}

func TestPushLocalCopyEmitsClassifiedUpdate(t *testing.T) {
	clip := newClip(&capturingWriter{})
	upd := make(chan eventful.Update, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = clip.Watch(ctx, upd) }()

	clip.PushLocalCopy("", []byte("hello world"))
	got := drainOne(t, upd)

	if !got.MimeType.IsText() {
		t.Fatalf("MimeType = %v, want text", got.MimeType)
	}
	if got.Size != uint64(len("hello world")) {
		t.Fatalf("Size = %d, want %d", got.Size, len("hello world"))
	}
	var dedup eventful.Deduplicator
	if got.Hash != dedup.Hash([]byte("hello world")) {
		t.Fatalf("Hash = %d, want xxhash of payload", got.Hash)
	}
}

func TestMimeHintOverridesSniff(t *testing.T) {
	clip := newClip(&capturingWriter{})
	upd := make(chan eventful.Update, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = clip.Watch(ctx, upd) }()

	// Plain-text bytes, but the host insists it is an image.
	clip.PushLocalCopy("image/png", []byte("not really an image"))
	got := drainOne(t, upd)

	if !got.MimeType.IsImage() {
		t.Fatalf("MimeType = %v, want image (hint override)", got.MimeType)
	}
}

func TestDuplicateLocalCopyIsDropped(t *testing.T) {
	clip := newClip(&capturingWriter{})
	upd := make(chan eventful.Update, 2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = clip.Watch(ctx, upd) }()

	clip.PushLocalCopy("", []byte("same"))
	_ = drainOne(t, upd)

	clip.PushLocalCopy("", []byte("same"))
	select {
	case u := <-upd:
		t.Fatalf("duplicate copy should be dropped, got %v", u)
	case <-time.After(150 * time.Millisecond):
	}
}

// A payload written from a peer must be marked as seen so that the host
// clipboard change it causes is not echoed back into the mesh.
func TestRemoteWriteSuppressesEcho(t *testing.T) {
	w := &capturingWriter{}
	clip := newClip(w)
	upd := make(chan eventful.Update, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = clip.Watch(ctx, upd) }()

	payload := []byte("from a peer")
	if _, err := clip.Write(mime.TypeText, payload); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if w.calls() != 1 {
		t.Fatalf("writer calls = %d, want 1", w.calls())
	}

	// The host clipboard listener now reports the very payload we injected.
	clip.PushLocalCopy("", payload)
	select {
	case u := <-upd:
		t.Fatalf("injected payload should not be re-broadcast, got %v", u)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestWriteWithoutWriterDrops(t *testing.T) {
	clip := newClip(nil)
	n, err := clip.Write(mime.TypeText, []byte("payload"))
	if err != nil {
		t.Fatalf("Write with nil writer should not error, got %v", err)
	}
	if n != len("payload") {
		t.Fatalf("n = %d, want %d", n, len("payload"))
	}
}

func TestWatchClosesChannelOnCancel(t *testing.T) {
	clip := newClip(&capturingWriter{})
	upd := make(chan eventful.Update)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		_ = clip.Watch(ctx, upd)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Watch did not return after cancel")
	}

	if _, open := <-upd; open {
		t.Fatal("Watch must close the update channel on exit")
	}
}
