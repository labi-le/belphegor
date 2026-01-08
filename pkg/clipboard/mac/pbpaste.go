//go:build unix

package mac

import (
	"context"

	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/mime"
)

var _ eventful.Eventful = &Clipboard{}

// todo buy mac and implement

type Clipboard struct{}

func (m *Clipboard) Watch(ctx context.Context, update chan<- eventful.Update) error {
	//TODO implement me
	panic("implement me")
}

func (m *Clipboard) Write(t mime.Type, src []byte) (int, error) {
	//TODO implement me
	panic("implement me")
}
