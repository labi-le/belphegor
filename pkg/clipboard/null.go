//go:build null

package clipboard

import (
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/clipboard/null"
	"github.com/rs/zerolog"
)

func New(opts eventful.Options, logger zerolog.Logger) *null.Clipboard {
	return null.NewNull()
}
