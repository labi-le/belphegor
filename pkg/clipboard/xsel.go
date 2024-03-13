package clipboard

import "os/exec"

type xSel struct{}

func (m *xSel) Set(data []byte) error {
	return clipboardSet(data,
		exec.Command("xsel", "--input", "--clipboard"),
	)
}

func (m *xSel) Get() ([]byte, error) {
	return clipboardGet(exec.Command("xsel", "--output", "--clipboard"))
}

func (m *xSel) Name() string {
	return XSel
}
