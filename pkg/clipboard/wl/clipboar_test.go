package wl_test

import (
	"context"
	"fmt"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/clipboard/wl"
	"testing"
)

func TestWlr_Watch(t *testing.T) {
	tests := []struct {
		name     string
		testData []byte
		wantErr  bool
	}{
		{
			name:     "Basic clipboard update",
			testData: []byte("test data"),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := wl.NewWlr()

			updates := make(chan clipboard.Update)
			go w.Watch(context.TODO(), updates)

			for update := range updates {
				fmt.Println(string(update.Data))
			}
		})
	}
}
