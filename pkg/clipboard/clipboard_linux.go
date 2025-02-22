//go:build linux

package clipboard

import (
	"errors"
	"github.com/labi-le/belphegor/pkg/clipboard/api"
	"github.com/labi-le/belphegor/pkg/clipboard/generic"
	"os/exec"
)

var ErrUnimplementedClipboardManager = errors.New("unimplemented clipboard wrapper")

var managers = []api.Manager{
	generic.XClip{},
	generic.XSel{},
	generic.WlClipboard{},
	generic.Termux{},
}

func findClipboardManager() api.Manager {
	for _, manager := range managers {
		_, err := manager.Get()

		if err == nil {
			return manager
		}

		var ee *exec.ExitError
		if errors.As(err, &ee) {
			if ee.ExitCode() == 1 {
				return manager
			}
		}
	}

	panic(ErrUnimplementedClipboardManager)
}

func New() api.Manager {
	return findClipboardManager()
}
