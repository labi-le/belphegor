package generic

import (
	"context"
	"github.com/labi-le/belphegor/pkg/clipboard/api"
	"os/exec"
	"time"
)

type Termux struct{}

func (m Termux) Watch(ctx context.Context, update chan<- api.Update) {
	for range time.After(2 * time.Second) {
		select {
		case <-ctx.Done():
			return
		default:
			output, err := exec.Command("termux-clipboard-get").Output()
			update <- api.Update{
				Data: output,
				Err:  err,
			}
		}
	}
}

func (m Termux) Write(p []byte) (n int, err error) {
	return len(p), ClipboardSet(p,
		exec.Command("termux-clipboard-set"),
	)
}
