//go:build debug

package node

import (
	"crypto/tls"
	"os"
	"path"

	"github.com/rs/zerolog"
)

func populateKeyLog(logger zerolog.Logger, conf *tls.Config) {

	const keysFilename = "belphegor-quic-keys.log"
	keyLogFile, err := os.OpenFile(
		path.Join(os.TempDir(), keysFilename),
		os.O_WRONLY|os.O_CREATE|os.O_APPEND,
		0600,
	)
	if err == nil {
		conf.KeyLogWriter = keyLogFile
	}
	logger.Debug().Str("path", keyLogFile.Name()).Msg("setup tls key log writer")
}
