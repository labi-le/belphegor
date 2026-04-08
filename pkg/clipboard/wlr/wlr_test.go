//go:build unix

package wlr_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/clipboard/wlr"
	"github.com/labi-le/belphegor/pkg/mime"
	"github.com/rs/zerolog"
)

func TestWatchContextCancel(t *testing.T) {
	if !wlr.Supported {
		t.Skip("Wayland not supported")
	}

	clip := wlr.Must(zerolog.New(nil), eventful.Options{})

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)

	upd := make(chan eventful.Update, 10)
	_ = clip.Watch(ctx, upd)
}

func TestWriteAndWatchTwoClients(t *testing.T) {
	if !wlr.Supported {
		t.Skip("Wayland not supported")
	}

	reader := wlr.Must(zerolog.Nop(), eventful.Options{})
	writer := wlr.Must(zerolog.Nop(), eventful.Options{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	upd := make(chan eventful.Update, 10)
	go func() { _ = reader.Watch(ctx, upd) }()
	go func() { _ = writer.Watch(ctx, make(chan eventful.Update, 10)) }()

	time.Sleep(300 * time.Millisecond)

	want := []byte("hello world")
	n, err := writer.Write(mime.TypeText, want)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(want) {
		t.Fatalf("Write returned %d, want %d", n, len(want))
	}

	select {
	case u := <-upd:
		if !bytes.Equal(u.Data, want) {
			// Watch might intercept data that was recorded before our test; drain
			if bytes.Equal((<-upd).Data, want) {
				return
			}
			t.Errorf("got %q, want %q", string(u.Data), string(want))
		}

	case <-time.After(3 * time.Second):
		t.Error("timeout waiting for data")
	}
}
