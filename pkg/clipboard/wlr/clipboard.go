package wlr

import (
	"context"
	wl "deedles.dev/wl/client"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/pipe"
	"os"
)

var Supported = (func() bool {
	_, exist1 := os.LookupEnv("WAYLAND_DISPLAY")
	_, exist2 := os.LookupEnv("WAYLAND_SOCKET")
	return exist1 || exist2
})()

type Wlr struct {
	reader *ClipboardReader
	writer *ClipboardWriter
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
		reader: NewClipboardReader(read, p),
		writer: NewClipboardWriter(write),
	}
}

func (w Wlr) Watch(ctx context.Context, update chan<- clipboard.Update) {
	go func() {
		<-ctx.Done()
		close(update)
	}()
	go w.reader.Run(ctx)

	for {
		buf, err := pipe.FromPipe(w.reader.Pipe.ReadFd())
		if err != nil {
			continue
		}

		update <- clipboard.Update{Data: buf, Err: err}
	}

}

func (w Wlr) Write(data []byte) (int, error) {
	return w.writer.Write(data)
}

func (w Wlr) Run(ctx context.Context) error {
	go w.writer.Run(ctx)

	<-ctx.Done()
	return ctx.Err()
}

func (w Wlr) Close() error {
	if err := w.writer.Close(); err != nil {
		return err
	}

	return w.reader.Close()
}
