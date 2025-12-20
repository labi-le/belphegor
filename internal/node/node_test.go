package node_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/labi-le/belphegor/internal/node"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/storage"
)

func TestNode_MessageExchange(t *testing.T) {
	testData := []byte("test message")
	ctx := context.TODO()

	clip1, node1, clip2, node2 := testNodes()

	go func() {
		node1.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	go func() {
		node2.Start(ctx)
	}()

	go func() {
		if err := node2.ConnectTo(ctx, "127.0.0.1:7777"); err != nil {
			t.Fatalf("failed to connect node2 to node1: %v", err)
		}
	}()

	if err := clip1.Set(testData); err != nil {
		t.Fatal(err)
	}

	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if data, err := clip2.Get(); err == nil {
				if data != nil && !bytes.Equal(data, testData) {
					t.Fatalf("expected: %s, actual: %s", testData, data)
				}
				return
			}
		case <-timeout:
			t.Fatal("timeout waiting for message")
		}
	}

}

func testNodes() (*clipboard.Null, *node.Node, *clipboard.Null, *node.Node) {
	clip1 := &clipboard.Null{}
	_ = clip1.Set([]byte("null"))

	node1 := node.New(
		clip1,
		storage.NewSyncMapStorage[domain.UniqueID, *node.Peer](),
		node.NewChannel(),
		node.WithPublicPort(7777),
		node.WithMetadata(domain.Device{
			Name: "1",
			Arch: "amd64",
			ID:   domain.NewID(),
		}),
	)

	clip2 := &clipboard.Null{}
	node2 := node.New(
		clip2,
		storage.NewSyncMapStorage[domain.UniqueID, *node.Peer](),
		node.NewChannel(),
		node.WithPublicPort(7778),
		node.WithMetadata(domain.Device{
			Name: "2",
			Arch: "amd64",
			ID:   domain.NewID(),
		}),
	)
	return clip1, node1, clip2, node2
}
