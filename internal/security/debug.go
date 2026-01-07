//go:build debug

package security

import (
	"crypto/tls"
	"io"
	"os"
	"path"
	"sync"

	"github.com/rs/zerolog"
)

var populateKeyLog = func() func(zerolog.Logger, *tls.Config) {
	var (
		once   sync.Once
		writer io.Writer
	)

	return func(logger zerolog.Logger, conf *tls.Config) {
		once.Do(func() {
			const keysFilename = "belphegor-quic-keys.log"
			f, err := os.OpenFile(
				path.Join(os.TempDir(), keysFilename),
				os.O_WRONLY|os.O_CREATE|os.O_APPEND,
				0600,
			)
			if err != nil {
				logger.Error().Err(err).Msg("failed to open key log file")
				return
			}

			writer = f
			logger.Debug().Str("path", f.Name()).Msg("setup tls key log writer")
		})

		if writer != nil {
			conf.KeyLogWriter = writer
		}
	}
}()
