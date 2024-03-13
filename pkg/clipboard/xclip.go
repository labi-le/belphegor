package clipboard

import "os/exec"

type xClip struct{}

func (m *xClip) Set(data []byte) error {
	return clipboardSet(data,
		exec.Command("xclip", "-in", "-selection", "clipboard"),
	)
}

func (m *xClip) Get() ([]byte, error) {
	return clipboardGet(exec.Command("xclip", "-out", "-selection", "clipboard"))
}

func (m *xClip) Name() string {
	return XClip
}
