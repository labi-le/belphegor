package eventful

import (
	"context"

	"github.com/labi-le/belphegor/pkg/mime"
)

type Eventful interface {
	// Watch subscribe to clipboard updates
	// when the context is finished, Watch must close the update channel
	Watch(ctx context.Context, update chan<- Update) error
	Write(p []byte) (n int, err error)
}

type Update struct {
	Data     []byte
	MimeType mime.Type
	Hash     uint64
}
