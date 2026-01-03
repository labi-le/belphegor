//go:build linux

package clipboard

import (
	"errors"
	"os/exec"

	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/clipboard/mac"
	"github.com/labi-le/belphegor/pkg/clipboard/wlclipboard"
	"github.com/labi-le/belphegor/pkg/clipboard/wlr"
	"github.com/rs/zerolog"
)

var ErrUnimplementedClipboardManager = errors.New("unimplemented clipboard wrapper")

const (
	WlClipboard   = "wl-copy"
	Mac           = "pbpaste"
	NullClipboard = "null-clipboard"
)

func findClipboardManager(logger zerolog.Logger) eventful.Eventful {
	if wlr.Supported {
		return wlr.Must(logger)
	}

	if commandExists(WlClipboard) {
		return new(wlclipboard.Clipboard)
	}

	if commandExists(Mac) {
		return new(mac.Clipboard)
	}

	panic(ErrUnimplementedClipboardManager)
}

func New(logger zerolog.Logger) eventful.Eventful {
	return findClipboardManager(logger)
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
