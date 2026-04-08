package x11

import (
	"testing"

	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/mime"
	"github.com/rs/zerolog"
)

func TestNew(t *testing.T) {
	logger := zerolog.Nop()
	opts := eventful.Options{}
	c := New(logger, opts)

	if c == nil {
		t.Fatal("expected non-nil Clipboard")
	}

	if c.opts != opts {
		t.Errorf("expected options to be set")
	}
}

func TestWrite_NotInitialized(t *testing.T) {
	c := New(zerolog.Nop(), eventful.Options{})
	_, err := c.Write(mime.TypeText, []byte("hello"))
	if err == nil || err.Error() != "x11 not initialized" {
		t.Errorf("expected error 'x11 not initialized', got %v", err)
	}
}
