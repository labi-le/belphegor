package generic

import (
	"context"
	"github.com/labi-le/belphegor/pkg/clipboard/api"
	"os/exec"
	"time"
)

type XClip struct{}

func (m XClip) Watch(ctx context.Context, update chan<- api.Update) {
	for range time.After(2 * time.Second) {
		select {
		case <-ctx.Done():
			return
		default:
			output, err := exec.Command("xclip", "-out", "-selection", "clipboard").Output()
			update <- api.Update{
				Data: output,
				Err:  err,
			}
		}
	}
}

func (m XClip) Write(p []byte) (n int, err error) {
	return len(p), ClipboardSet(p,
		exec.Command("xclip", "-in", "-selection", "clipboard"),
	)
}
