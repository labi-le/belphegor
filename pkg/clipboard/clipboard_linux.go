//go:build linux

package clipboard

import (
	"errors"
	"os/exec"
)

var ErrUnimplementedClipboardManager = errors.New("unimplemented clipboard wrapper")

var managers = map[string]Manager{
	XClip:         new(xClip),
	XSel:          new(xSel),
	WlClipboard:   new(wlClipboard),
	Termux:        new(termux),
	NullClipboard: new(Null),
}

func findClipboardManager() Manager {
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

func New() Manager {
	return &wrapped{
		findClipboardManager(),
	}
}

type wrapped struct {
	manager Manager
}

func (w *wrapped) Set(data []byte) error {
	return w.manager.Set(data)
}

func (w *wrapped) Get() ([]byte, error) {
	return w.manager.Get()
}

func (w *wrapped) Name() string {
	return w.manager.Name()
}
