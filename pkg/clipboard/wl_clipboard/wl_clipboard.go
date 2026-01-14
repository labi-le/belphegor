package wl_clipboard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync/atomic"
	"time"

	"github.com/cespare/xxhash"
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/mime"
	"github.com/rs/zerolog"
)

var _ eventful.Eventful = (*Clipboard)(nil)

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
	logger   zerolog.Logger
	lastHash atomic.Uint64
}

func New(log zerolog.Logger) *Clipboard {
	return &Clipboard{
		logger: log.With().Str("component", "wl_clipboard").Logger(),
	}
}

func (c *Clipboard) Watch(ctx context.Context, upd chan<- eventful.Update) error {
	defer close(upd)

	ticker := time.NewTicker(clipboardTick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			get, err := clipboardGet(exec.Command("wl-paste", "--no-newline"))
			if err != nil {
				var err2 *exec.ExitError
				// usually this means the buffer is empty
				if errors.As(err, &err2) && err2.ExitCode() == 1 {
					continue
				}
				c.logger.Error().Err(err).Msg("wl-clipboard: failed to get content")
				continue
			}

			if len(get) == 0 {
				continue
			}

			if !c.dedup(get) {
				continue
			}

			upd <- eventful.Update{
				Data:     get,
				MimeType: mime.From(get),
				Hash:     c.lastHash.Load(),
			}
		}
	}
}

func (c *Clipboard) Write(_ mime.Type, src []byte) (int, error) {
	dataHash := xxhash.Sum64(src)
	c.lastHash.Store(dataHash)

	if err := clipboardSet(src, exec.Command("wl-copy")); err != nil {
		c.logger.Error().Err(err).Msg("failed to write to wl-clipboard")
		return 0, err
	}

	return len(src), nil
}

func (c *Clipboard) dedup(data []byte) bool {
	dataHash := xxhash.Sum64(data)

	if dataHash == c.lastHash.Load() {
		return false
	}

	c.lastHash.Store(dataHash)
	return true
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
