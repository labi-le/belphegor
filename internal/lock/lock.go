package lock

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/nightlyone/lockfile"
	"github.com/rs/zerolog"
)

const file = "belphegor.lck"

var (
	ErrCannotLock     = errors.New("cannot get locked process: %s")
	ErrCannotUnlock   = errors.New("cannot unlock process: %s")
	ErrAlreadyRunning = errors.New("belphegor is already running. pid %d")
)

func Must(logger zerolog.Logger) func() {
	lock, _ := lockfile.New(filepath.Join(os.TempDir(), file))

	if lockErr := lock.TryLock(); lockErr != nil {
		owner, err := lock.GetOwner()
		if err != nil {
			logger.Fatal().Msgf(ErrCannotLock.Error(), err)
		}
		logger.Fatal().Msgf(ErrAlreadyRunning.Error(), owner.Pid)
	}

	return func() {
		Unlock(lock, logger)
	}
}

func Unlock(lock lockfile.Lockfile, l zerolog.Logger) {
	if err := lock.Unlock(); err != nil {
		l.Fatal().Msgf(ErrCannotUnlock.Error(), err)
	}
}
