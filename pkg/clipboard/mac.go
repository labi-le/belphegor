//go:build darwin && !null

package clipboard

import (
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/clipboard/mac"
	"github.com/rs/zerolog"
)

func New(logger zerolog.Logger, opts eventful.Options) *mac.Clipboard {
	return mac.New(logger, opts)
}
