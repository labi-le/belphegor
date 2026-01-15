//go:build unix && !darwin && !wl_clipboard && !null && !x11

package clipboard

import (
	wl "deedles.dev/wl/client"
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/clipboard/wlr"
	"github.com/rs/zerolog"
)

func New(logger zerolog.Logger) eventful.Eventful {
	client, err := wl.Dial()
	if err != nil {
		logger.Fatal().Err(err).Send()
	}

	return wlr.New(client, logger)
}
