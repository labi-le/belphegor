//go:build windows
// +build windows

package clipboard

import (
	"os/exec"
)

type ClipboardManager interface {
	Get() ([]byte, error)
	Set(data []byte) error
	Exist() bool
}

func NewManager() Manager {
	return &windows{}

}

type windows struct{}

func (p windows) Get() ([]byte, error) {
	return clipboardGet(exec.Command("powershell.exe", "-command", "Get-Clipboard"))
}

func (p windows) Set(data []byte) error {
	return clipboardSet(data, exec.Command("clip"))
}
