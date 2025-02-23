package generic

import (
	"context"
	"github.com/labi-le/belphegor/pkg/clipboard/api"
	"time"
)

type Null struct {
	data []byte
}

func (n *Null) Watch(ctx context.Context, update chan<- api.Update) {
	for range time.After(2 * time.Second) {
		select {
		case <-ctx.Done():
			return
		default:
			update <- api.Update{Data: n.data}
		}
	}
}

func (n *Null) Write(p []byte) (int, error) {
	n.data = p
	return len(p), nil
}
