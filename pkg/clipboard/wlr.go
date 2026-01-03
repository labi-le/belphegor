//go:build unix && !darwin && !wl_clipboard && !null

package clipboard

import (
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/clipboard/wlr"
	"github.com/rs/zerolog"
)

func New(logger zerolog.Logger) eventful.Eventful {
	return wlr.Must(logger)
}
