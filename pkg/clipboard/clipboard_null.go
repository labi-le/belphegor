package clipboard

import (
	"bytes"
	"context"
)

type Null struct {
	data       []byte
	RootUpdate chan<- Update
}

func (n *Null) Watch(ctx context.Context, _ chan<- Update) error {
	current := n.data
	for {
		select {
		case <-ctx.Done():
		default:
			if !bytes.Equal(current, n.data) {
				current = n.data
				n.RootUpdate <- Update{Data: current}
			}
		}
	}
}

func (n *Null) Write(p []byte) (int, error) {
	n.data = p
	return len(n.data), nil
}

func (n *Null) Get() ([]byte, error) {
	return n.data, nil
}

func (n *Null) Set(data []byte) error {
	n.data = data
	return nil
}

func (n *Null) Name() string {
	return NullClipboard
}
