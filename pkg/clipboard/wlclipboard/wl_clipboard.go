package wlclipboard

import (
	"context"

	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
)

type Clipboard struct{}

func (m *Clipboard) Watch(ctx context.Context, update chan<- eventful.Update) error {
	//TODO implement me
	panic("implement me")
}

func (m *Clipboard) Write(p []byte) (n int, err error) {
	//TODO implement me
	panic("implement me")
}
