package node_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
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
		{
			name: "10mb",
			msg:  []byte(strings.Repeat("a", 10<<20)),
		},
	}

	initConn, clip1, clip2 := testNodes(t)
	go initConn(t.Context())

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			time.Sleep(50 * time.Millisecond)

			go clip1.Write(tc.msg)

			for {
				select {
				case data := <-clip2.RootUpdate:
					if !bytes.Equal(data, tc.msg) {
						t.Fatalf("expected: %s, actual: %s", tc.msg, data)
					}
					_, _ = clip2.Write(nil)
					return
				case <-time.After(100 * time.Second):
					t.Fatal("timeout waiting for message")
				}
			}
		})
	}
}

func testNodes(t testing.TB) (func(ctx context.Context), *clipboard.Null, *clipboard.Null) {
	clip1 := clipboard.NewNull()
	go clip1.Write([]byte("a"))

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
		node.WithClipboardScanDelay(100*time.Millisecond),
	)

	clip2 := clipboard.NewNull()

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

//func BenchmarkNode_MessageExchange(b *testing.B) {
//	log.Logger = zerolog.Nop()
//
//	benchmarks := []struct {
//		name string
//		size int
//	}{
//		{"1KB", 1024},
//		//{"64KB", 64 * 1024},
//		//{"512KB", 512 * 1024},
//		//{"1MB", 1024 * 1024},
//	}
//
//	initConn, clip1, clip2 := testNodes(b)
//	go initConn(b.Context())
//
//	for _, bm := range benchmarks {
//		b.Run(bm.name, func(b *testing.B) {
//			time.Sleep(50 * time.Millisecond)
//
//			timeout := time.After(10 * time.Second)
//			ticker := time.NewTicker(100 * time.Millisecond)
//			defer ticker.Stop()
//
//			payload := make([]byte, bm.size)
//
//			b.ReportAllocs()
//			b.ResetTimer() // Сбрасываем таймер и счетчик аллокаций
//
//			for i := 0; i < b.N; i++ {
//				payload[0] = byte(i)
//
//				_, _ = clip1.Write(payload)
//
//				done := false
//				for !done {
//					select {
//					case <-ticker.C:
//						data, _ := clip2.RootUpdate()
//						if data == nil {
//							continue
//						}
//
//						if len(data) != len(payload) {
//							b.Fatal("error")
//						}
//
//						done = true
//						continue
//					case <-timeout:
//						b.Fatal("timeout waiting for message")
//					}
//				}
//			}
//		})
//	}
//
//}
