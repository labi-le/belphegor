//go:build unix && x11 && !darwin && !wl_clipboard && !null

package clipboard

import (
	"os"

	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/clipboard/x11"
	"github.com/rs/zerolog"
)

func New(opts eventful.Options, logger zerolog.Logger) *x11.Clipboard {
	if _, ok := os.LookupEnv("DISPLAY"); !ok {
		logger.Fatal().Msg("x11 display not found")
	}
	return x11.New(logger, opts)
}
