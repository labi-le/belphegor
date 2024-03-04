//go:build darwin

package clipboard

import "os/exec"

func NewManager() Manager {
	return &darwin{}
}

type darwin struct{}

func (p darwin) Get() ([]byte, error) {
	return clipboardGet(exec.Command("pbpaste"))
}

func (p darwin) Set(data []byte) error {
	return clipboardSet(data, exec.Command("pbcopy"))
}
