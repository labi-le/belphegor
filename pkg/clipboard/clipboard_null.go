package clipboard

import (
	"context"
)

type Null struct {
	data       chan []byte
	RootUpdate chan []byte
}

func NewNull() *Null {
	return &Null{RootUpdate: make(chan []byte, 1), data: make(chan []byte, 2)}
}

func (n *Null) Watch(_ context.Context, up chan<- Update) error {
	for data := range n.data {
		up <- Update{Data: data}
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

func (n *Null) Write(p []byte) (int, error) {
	n.data <- p

	return len(n.data), nil
}

func (n *Null) Name() string {
	return NullClipboard
}
