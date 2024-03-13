package clipboard

import "os/exec"

type wlClipboard struct{}

func (m *wlClipboard) Set(data []byte) error {
	return clipboardSet(data,
		exec.Command("wl-copy"),
	)
}

func (m *wlClipboard) Get() ([]byte, error) {
	return clipboardGet(exec.Command("wl-paste", "--no-newline"))
}

func (m *wlClipboard) Name() string {
	return WlClipboard
}
