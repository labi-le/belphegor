package generic

import (
	"os/exec"
)

type Termux struct{}

func (m Termux) Set(data []byte) error {
	return ClipboardSet(data,
		exec.Command("termux-clipboard-set"),
	)
}

func (m Termux) Get() ([]byte, error) {
	return ClipboardGet(exec.Command("termux-clipboard-get"))
}
