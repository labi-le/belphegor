package wl

import (
	"context"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/pipe"
	"log"
)

type Wlr struct {
	state state
}

func NewWlr() *Wlr {
	w := &Wlr{state{pipe: pipe.MustNonBlock()}}
	if err := w.state.init(); err != nil {
		log.Fatalf("init: %v", err)
	}

	return w
}

func (w *Wlr) Watch(ctx context.Context, update chan<- clipboard.Update) {
	go func() {
		<-ctx.Done()
		close(update)
	}()
	go func() {
		for {
			buf, err := pipe.FromPipe(w.state.pipe.ReadFd())
			if err != nil {
				continue
			}

			update <- clipboard.Update{Data: buf}
		}

	}()

	defer w.state.client.Close()
	w.state.run(ctx)
}
