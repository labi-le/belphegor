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
}

func NewNull() *Clipboard {
	return &Clipboard{RootUpdate: make(chan []byte), data: make(chan []byte)}
}

func (n *Clipboard) Watch(_ context.Context, upd chan<- eventful.Update) error {
	defer close(upd)

	for data := range n.data {
		upd <- eventful.Update{Data: data}
		n.RootUpdate <- data
		//select {
		//case up <- Update{Data: data}:
		//default:
		//}
		//
		//select {
		//case n.RootUpdate <- data:
		//default:
		//}
	}

	return nil
}

func (n *Clipboard) Write(_ mime.Type, src []byte) (int, error) {
	n.data <- src

	return len(n.data), nil
}
