package eventful

import (
	"context"

	"github.com/labi-le/belphegor/pkg/mime"
	"github.com/rs/zerolog"
)

type Eventful interface {
	// Watch subscribe to clipboard updates
	// when the context is finished, Watch must close the update channel
	Watch(ctx context.Context, upd chan<- Update) error
	Write(t mime.Type, src []byte) (int, error)
}

type Update struct {
	Data     []byte
	MimeType mime.Type
	Hash     uint64
}

func (u Update) MarshalZerologObject(e *zerolog.Event) {
	e.Int("length", len(u.Data))
	e.Uint64("hash", u.Hash)
	e.Stringer("mime", u.MimeType)
}
