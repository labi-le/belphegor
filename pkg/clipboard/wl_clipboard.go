//go:build unix && !darwin && !null && wl_clipboard

package clipboard

import (
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/clipboard/wl_clipboard"
	"github.com/rs/zerolog"
)

func New(opts eventful.Options, logger zerolog.Logger) *wl_clipboard.Clipboard {
	return wl_clipboard.New(logger)
}
