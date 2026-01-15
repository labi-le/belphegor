package null

import (
	"context"

	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/mime"
)

var _ eventful.Eventful = (*Clipboard)(nil)

type Clipboard struct {
	data       chan []byte
	RootUpdate chan []byte
	dedup      eventful.Deduplicator
}

func NewNull() *Clipboard {
	return &Clipboard{RootUpdate: make(chan []byte), data: make(chan []byte)}
}

func (n *Clipboard) Watch(_ context.Context, upd chan<- eventful.Update) error {
	defer close(upd)

	for data := range n.data {
		if h, ok := n.dedup.Check(data); ok {
			upd <- eventful.Update{
				Data:     data,
				MimeType: mime.From(data),
				Hash:     h,
			}
			n.RootUpdate <- data
		}
	}

	return nil
}

func (n *Clipboard) Write(_ mime.Type, src []byte) (int, error) {
	n.dedup.Mark(src)
	n.data <- src

	return len(n.data), nil
}
