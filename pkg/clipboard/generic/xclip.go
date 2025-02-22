package generic

import (
	"os/exec"
)

type XClip struct{}

func (m XClip) Set(data []byte) error {
	return ClipboardSet(data,
		exec.Command("xclip", "-in", "-selection", "clipboard"),
	)
}

func (m XClip) Get() ([]byte, error) {
	return ClipboardGet(exec.Command("xclip", "-out", "-selection", "clipboard"))
}
