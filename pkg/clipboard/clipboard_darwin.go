//go:build darwin

package clipboard

import "os/exec"

func NewManager() *Darwin {
	return &Darwin{}
}

type Darwin struct{}

func (p *Darwin) Get() ([]byte, error) {
	return clipboardGet(exec.Command("pbpaste"))
}

func (p *Darwin) Set(data []byte) error {
	return clipboardSet(data, exec.Command("pbcopy"))
}

func (p *Darwin) Name() string {
	return MasOsStd
}
