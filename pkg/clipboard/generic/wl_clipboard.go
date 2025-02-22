package generic

import (
	"os/exec"
)

type WlClipboard struct{}

func (m WlClipboard) Set(data []byte) error {
	return ClipboardSet(data,
		exec.Command("wl-copy"),
	)
}

func (m WlClipboard) Get() ([]byte, error) {
	return ClipboardGet(exec.Command("wl-paste", "--no-newline"))
}
