package wl_clipboard

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/mime"
)

var clipboardTick = 3 * time.Second

func init() {
	tick, exist := os.LookupEnv("WL_CLIPBOARD_TICK")
	if !exist {
		return
	}
	duration, err := time.ParseDuration(tick)
	if err != nil {
		panic(fmt.Sprintf("failed value for WL_CLIPBOARD_TICK. example: 5s"))
	}

	clipboardTick = duration
}

type Clipboard struct {
	mu   sync.Mutex
	last []byte
}

func (m *Clipboard) Watch(ctx context.Context, update chan<- eventful.Update) error {
	defer close(update)

	for range time.Tick(clipboardTick) {
		if ctx.Err() != nil {
			return nil
		}

		get, err := clipboardGet(exec.Command("wl-paste", "--no-newline"))
		if err != nil {
			var err2 *exec.ExitError
			// usually this means the buffer is empty
			if errors.As(err, &err2) && err2.ExitCode() == 1 {
				continue
			}
			return fmt.Errorf("wl-clipboard: %w", err)
		}

		m.mu.Lock()
		if bytes.Equal(m.last, get) {
			m.mu.Unlock()
			continue
		}
		m.last = get
		m.mu.Unlock()

		update <- eventful.Update{
			Data:     get,
			MimeType: mime.From(get),
		}
	}

	return nil
}

func (m *Clipboard) Write(p []byte) (n int, err error) {
	if err := clipboardSet(p, exec.Command("wl-copy")); err != nil {
		return 0, err
	}

	m.mu.Lock()
	m.last = make([]byte, len(p))
	copy(m.last, p)
	m.mu.Unlock()

	return len(p), nil
}

func clipboardGet(cmd *exec.Cmd) ([]byte, error) {
	return cmd.Output()
}

func clipboardSet(data []byte, cmd *exec.Cmd) error {
	var (
		in  io.WriteCloser
		err error
	)

	if in, err = cmd.StdinPipe(); err != nil {
		return err
	}

	if err = cmd.Start(); err != nil {
		return err
	}

	if _, err = in.Write(data); err != nil {
		return err
	}

	if err = in.Close(); err != nil {
		return err
	}

	return cmd.Wait()
}
