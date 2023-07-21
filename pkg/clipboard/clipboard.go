package clipboard

import (
	"github.com/rs/zerolog/log"
	"io"
	"os/exec"
)

type Manager interface {
	Get() ([]byte, error)
	Set(data []byte) error
}

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

	go func() {
		if err = cmd.Wait(); err != nil {
			log.Error().Err(err).Msg("clipboardSet")
		}
	}()
	return err
}

func toolExist(tool string) bool {
	_, err := exec.LookPath(tool)
	return err == nil
}
