package clipboard

import (
	"context"
)

type Eventful interface {
	Watch(ctx context.Context, update chan<- Update) error
	Write(p []byte) (n int, err error)
}

type Update struct {
	Data []byte
}
