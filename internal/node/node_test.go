package node_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/labi-le/belphegor/internal/node"
	"github.com/labi-le/belphegor/internal/notification"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/id"
)

func TestNode_MessageExchange(t *testing.T) {
	type testCase struct {
		name string
		msg  []byte
	}

	testCases := []testCase{
		{
			name: "short text",
			msg:  []byte("hello world"),
		},
		{
			name: "sentence with punctuation",
			msg:  []byte("this is a sample message, with commas, dots... ok?"),
		},
		{
			name: "multiline text",
			msg:  []byte("first line\nsecond line\nthird line"),
		},
		{
			name: "unicode symbols",
			msg:  []byte("text with unicode ✓ © ™ ∑ €"),
		},
		{
			name: "long message",
			msg:  []byte("this is a longer test message used to verify that larger payloads are transmitted correctly between nodes"),
		},
		{
			name: "mixed content",
			msg:  []byte("user john_doe42 connected from 192.168.0.15 at 10:23:59"),
		},
		{
			name: "json like payload",
			msg:  []byte(`{"event":"clipboard_update","value":"random english text","ok":true}`),
		},
	}

	initConn, clip1, clip2 := testNodes(t)
	go initConn(t.Context())

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			time.Sleep(50 * time.Millisecond)

			_ = clip1.Set(tc.msg)

			timeout := time.After(3 * time.Second)
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					data, _ := clip2.Get()
					if data == nil {
						continue
					}

					if !bytes.Equal(data, tc.msg) {
						t.Fatalf("expected: %s, actual: %s", tc.msg, data)
					}
					_ = clip2.Set(nil)
					return
				case <-timeout:
					t.Fatal("timeout waiting for message")
				}
			}
		})
	}
}

func testNodes(t *testing.T) (func(ctx context.Context), *clipboard.Null, *clipboard.Null) {
	clip1 := new(clipboard.Null)
	_ = clip1.Set([]byte("a"))

	port1 := 7777
	port2 := 7778
	addr := fmt.Sprintf("127.0.0.1:%d", port1)

	node1 := node.New(
		clip1,
		new(node.Storage),
		node.NewChannel(),
		node.WithPublicPort(port1),
		node.WithNotifier(notification.NullNotifier{}),
		node.WithMetadata(domain.Device{
			Name: "1",
			Arch: "amd64",
			ID:   id.New(),
		}),
	)

	clip2 := new(clipboard.Null)

	node2 := node.New(
		clip2,
		new(node.Storage),
		node.NewChannel(),
		node.WithPublicPort(port2),
		node.WithNotifier(notification.NullNotifier{}),
		node.WithMetadata(domain.Device{
			Name: "2",
			Arch: "amd64",
			ID:   id.New(),
		}),
	)

	initConn := func(ctx context.Context) {
		go node1.Start(ctx)
		go node2.Start(ctx)

		time.Sleep(100 * time.Millisecond)

		if err := node2.ConnectTo(
			ctx,
			addr,
		); err != nil {
			t.Fatalf("failed to connect node2 to node1: %v", err)
		}
	}

	return initConn, clip1, clip2
}
