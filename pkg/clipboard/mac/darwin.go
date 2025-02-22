//go:build darwin

package mac

import (
	"github.com/labi-le/belphegor/pkg/clipboard/generic"
	"os/exec"
)

func New() *Darwin {
	return &Darwin{}
}

type Darwin struct{}

func (p *Darwin) Get() ([]byte, error) {
	return generic.ClipboardGet(exec.Command("pbpaste"))
}

func (p *Darwin) Set(data []byte) error {
	return generic.ClipboardSet(data, exec.Command("pbcopy"))
}
