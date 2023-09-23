//go:build linux
// +build linux

package clipboard

import (
	"errors"
	"os/exec"
)

var ErrUnimplementedClipboardManager = errors.New("unimplemented clipboard wrapper")

var managers = []Manager{
	xClip{},
	xSel{},
	wlClipboard{},
	termux{},
}

func foundClipboardManager() Manager {

	for _, manager := range managers {
		switch manager.(type) {
		case xClip:
			if toolExist("xclip") {
				return manager
			}
			continue

		case xSel:
			if toolExist("xsel") {
				return manager
			}
			continue

		case wlClipboard:
			if toolExist("wl-copy") {
				return manager
			}
			continue

		case termux:
			if toolExist("termux-clipboard-set") {
				return manager
			}
		}
	}

	panic(ErrUnimplementedClipboardManager)
}

func NewManager() Manager {
	return &wrapped{
		foundClipboardManager(),
	}
}

type wrapped struct {
	manager Manager
}

func (w *wrapped) Get() ([]byte, error) {
	return w.manager.Get()
}

func (w *wrapped) Set(data []byte) error {
	return w.manager.Set(data)
}

type xClip struct{}

func (m xClip) Set(data []byte) error {
	return clipboardSet(data,
		exec.Command("xclip", "-in", "-selection", "clipboard"),
	)
}

func (m xClip) Get() ([]byte, error) {
	return clipboardGet(exec.Command("xclip", "-out", "-selection", "clipboard"))
}

type xSel struct{}

func (m xSel) Set(data []byte) error {
	return clipboardSet(data,
		exec.Command("xsel", "--input", "--clipboard"),
	)
}

func (m xSel) Get() ([]byte, error) {
	return clipboardGet(exec.Command("xsel", "--output", "--clipboard"))
}

type wlClipboard struct{}

func (m wlClipboard) Set(data []byte) error {
	return clipboardSet(data,
		exec.Command("wl-copy"),
	)
}

func (m wlClipboard) Get() ([]byte, error) {
	return clipboardGet(exec.Command("wl-paste", "--no-newline"))
}

type termux struct{}

func (m termux) Set(data []byte) error {
	return clipboardSet(data,
		exec.Command("termux-clipboard-set"),
	)
}

func (m termux) Get() ([]byte, error) {
	return clipboardGet(exec.Command("termux-clipboard-get"))
}
