package wlr

import (
	"context"
	wl "deedles.dev/wl/client"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/pipe"
)

type Wlr struct {
	R *ClipboardReader
	W *ClipboardWriter
}

func Must() Wlr {
	r, err := wl.Dial()
	if err != nil {
		panic(err)
	}
	w, err := wl.Dial()
	if err != nil {
		panic(err)
	}
	return New(r, w, pipe.MustNonBlock())
}

func New(read *wl.Client, write *wl.Client, p pipe.RWPipe) Wlr {
	return Wlr{
		R: NewClipboardReader(read, p),
		W: NewClipboardWriter(write),
	}
}

func (w Wlr) Watch(ctx context.Context, update chan<- clipboard.Update) {
	go func() {
		<-ctx.Done()
		close(update)
	}()
	defer w.W.Close()
	for {
		buf, err := pipe.FromPipe(w.R.Pipe.ReadFd())
		if err != nil {
			continue
		}

		update <- clipboard.Update{Data: buf}
	}

}

func (w Wlr) Write(data []byte) (int, error) {
	return w.W.Write(data)
}

func (w Wlr) Run(ctx context.Context) error {
	go w.R.Run(ctx)
	go w.W.Run(ctx)

	<-ctx.Done()
	return ctx.Err()
}
