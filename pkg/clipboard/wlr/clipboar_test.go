package wlr_test

import (
	"bytes"
	"context"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/clipboard/wlr"
	"math/rand"
	"testing"
	"time"
)

func generateRandomData(length int) []byte {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	data := make([]byte, length)
	for i := range data {
		data[i] = charset[rand.Intn(len(charset))]
	}
	return data
}

type testcase struct {
	name    string
	data    []byte
	wantErr bool
}

func TestWlr_WatchWrite(t *testing.T) {
	tests := []testcase{
		{"Basic clipboard update", []byte("test data\n"), false},
		{"Empty clipboard", []byte(""), true},
		{"Single character", []byte("a"), false},
		{"Long string", generateRandomData(10000), false},
		{"UTF-8 characters", []byte("こんにちは, 世界!"), false},
		{"Random length string", generateRandomData(rand.Intn(1000)), false},
	}

	//log.Logger = zerolog.Nop()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			w := wlr.Must()
			go w.Run(ctx)

			// warmup
			<-time.After(time.Millisecond)

			written := make(chan struct{})
			go func() {
				_, err := w.Write(tt.data)
				if err != nil {
					t.Error(err)
					return
				}

				written <- struct{}{}
			}()

			select {
			case <-written:
				checkUpdate(ctx, cancel, t, tt)
				return
			case <-ctx.Done():
				t.Error("failed to check clipboard functionality")
				return
			}
		})

	}
}

func checkUpdate(ctx context.Context, cancel context.CancelFunc, t *testing.T, test testcase) {
	updates := make(chan clipboard.Update)
	go wlr.Must().Watch(ctx, updates)

	select {
	case update := <-updates:
		if update.Err != nil {
			t.Errorf("watch error: %v", update.Err)
			return
		}
		if !bytes.Equal(update.Data, test.data) {
			t.Errorf("failed to validate data, got: %q, expect: %q", update.Data, test.data)
			return
		}
		cancel()
	case <-ctx.Done():
		if test.wantErr {
			return
		}
		t.Error("failed to validate data: timeout reached")
	}
}
