package generic

import (
	"os/exec"
)

type XSel struct{}

func (m XSel) Set(data []byte) error {
	return ClipboardSet(data,
		exec.Command("xsel", "--input", "--clipboard"),
	)
}

func (m XSel) Get() ([]byte, error) {
	return ClipboardGet(exec.Command("xsel", "--output", "--clipboard"))
}
