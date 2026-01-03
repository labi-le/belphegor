package null

import (
	"context"

	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
)

type Clipboard struct {
	data       chan []byte
	RootUpdate chan []byte
}

func NewNull() *Clipboard {
	return &Clipboard{RootUpdate: make(chan []byte), data: make(chan []byte)}
}

func (n *Clipboard) Watch(_ context.Context, up chan<- eventful.Update) error {
	defer close(up)

	for data := range n.data {
		up <- eventful.Update{Data: data}
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

func (n *Clipboard) Write(p []byte) (int, error) {
	n.data <- p

	return len(n.data), nil
}
