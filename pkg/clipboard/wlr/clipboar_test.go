package wlr_test

import (
	"context"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/clipboard/wlr"
	"testing"
	"time"
)

func TestWlr_Watch(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "Basic clipboard update",
			wantErr: false,
		},
	}

	w := wlr.Must()
	go w.Run(context.TODO())
	time.Sleep(2 * time.Second)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()
			updates := make(chan clipboard.Update)
			go w.Watch(ctx, updates)

			select {
			case <-ctx.Done():
				t.Error("failed to check clipboard functionality")
				return
			case update := <-updates:
				if update.Err != nil {
					t.Error(update.Err)
				}

				if len(update.Data) == 0 {
					t.Error(update.Err)
				}

				return
			}
		})
	}
}

func TestWlr_Write(t *testing.T) {
	//tests := []struct {
	//	name    string
	//	wantErr bool
	//}{
	//	{
	//		name:    "Basic clipboard update",
	//		wantErr: false,
	//	},
	//}

	//log.Logger = zerolog.Nop()

	w := wlr.Must()
	go w.Run(context.TODO())
	time.Sleep(2 * time.Second)
	_, err := w.Write([]byte("test data\n"))
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(40 * time.Second)
	//for _, tt := range tests {
	//	t.Run(tt.name, func(t *testing.T) {
	//		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	//		defer cancel()
	//
	//		written := make(chan struct{})
	//		go func() {
	//			n, err := w.Write([]byte("test data\n"))
	//			if err != nil {
	//				t.Error(err)
	//				return
	//			}
	//
	//			t.Logf("written: %d", n)
	//			written <- struct{}{}
	//
	//		}()
	//
	//		select {
	//		case <-written:
	//			break
	//		case <-ctx.Done():
	//			t.Error("failed to check clipboard functionality")
	//			return
	//
	//		}
	//	})
	//}
}
