package clipboard

import (
	"io"
	"os/exec"
)

type Manager interface {
	Get() ([]byte, error)
	Set(data []byte) error
	Name() string
}

const (
	XClip         = "xclip"
	XSel          = "xsel"
	WlClipboard   = "wl-clipboard"
	Termux        = "termux"
	WindowsNT10   = "nt10"
	MasOsStd      = "masos-std"
	NullClipboard = "null-clipboard"
)

func clipboardGet(cmd *exec.Cmd) ([]byte, error) {
	return cmd.Output()
}

func clipboardSet(data []byte, cmd *exec.Cmd) error {
	var (
		in  io.WriteCloser
		err error
	)

	if in, err = cmd.StdinPipe(); err != nil {
		return err
	}

	if err = cmd.Start(); err != nil {
		return err
	}

	if _, err = in.Write(data); err != nil {
		return err
	}

	if err = in.Close(); err != nil {
		return err
	}

	return cmd.Wait()
}
