//go:build linux

package clipboard

import (
	"errors"
)

var ErrUnimplementedClipboardManager = errors.New("unimplemented clipboard wrapper")

var managers = map[string]Manager{
	XClip:       new(xClip),
	XSel:        new(xSel),
	WlClipboard: new(wlClipboard),
	Termux:      new(termux),
}

func findClipboardManager() Manager {
	for _, manager := range managers {
		if _, err := manager.Get(); err == nil {
			return manager
		}
	}

	panic(ErrUnimplementedClipboardManager)
}

func NewManager() Manager {
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
