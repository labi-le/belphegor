//go:build null

package clipboard

import (
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/clipboard/null"
	"github.com/rs/zerolog"
)

func New(logger zerolog.Logger, opts eventful.Options) *null.Clipboard {
	return null.New(logger, opts)
}
