//go:build darwin && !null

package clipboard

import (
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/clipboard/mac"
	"github.com/rs/zerolog"
)

func New(opts eventful.Options, logger zerolog.Logger) *mac.Clipboard {
	return mac.New(opts)
}
