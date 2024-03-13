package clipboard

import "os/exec"

type termux struct{}

func (m *termux) Set(data []byte) error {
	return clipboardSet(data,
		exec.Command("termux-clipboard-set"),
	)
}

func (m *termux) Get() ([]byte, error) {
	return clipboardGet(exec.Command("termux-clipboard-get"))
}

func (m *termux) Name() string {
	return Termux
}
