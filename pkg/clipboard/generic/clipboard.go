package generic

import (
	"io"
	"os/exec"
)

func ClipboardGet(cmd *exec.Cmd) ([]byte, error) {
	return cmd.Output()
}

func ClipboardSet(data []byte, cmd *exec.Cmd) error {
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
