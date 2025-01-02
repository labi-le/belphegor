package clipboard

import "context"

type Eventful interface {
	Watch(ctx context.Context, update chan<- Update)
}

type Update struct {
	Data []byte
	Err  error
}
