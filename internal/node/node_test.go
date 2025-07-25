package node_test

import (
	"bytes"
	"context"
	"github.com/labi-le/belphegor/internal/node"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/storage"
	"testing"
	"time"
)

func TestNode_MessageExchange(t *testing.T) {
	testData := []byte("test message")
	ctx := context.TODO()

	clip1, node1, clip2, node2 := testNodes()

	go func() {
		node1.Start(context.TODO())
	}()

	time.Sleep(100 * time.Millisecond)

	go func() {
		node2.Start(context.TODO())
	}()

	go func() {
		if err := node2.ConnectTo(ctx, "127.0.0.1:7777"); err != nil {
			t.Fatalf("failed to connect node2 to node1: %v", err)
		}
	}()

	if err := clip1.Set(testData); err != nil {
		t.Fatal(err)
	}

	deadline := time.After(3 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for message")
		default:
			if data, err := clip2.Get(); err == nil {
				if bytes.Equal(data, testData) {
					return
				}
			}
			time.Sleep(100 * time.Millisecond)
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
		node.WithMetadata(domain.MetaData{
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
		node.WithMetadata(domain.MetaData{
			Name: "2",
			Arch: "amd64",
			ID:   domain.NewID(),
		}),
	)
	return clip1, node1, clip2, node2
}
