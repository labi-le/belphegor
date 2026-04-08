//go:build windows

package clipboard

import (
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/clipboard/windows"
	"github.com/rs/zerolog"
)

func New(logger zerolog.Logger, opts eventful.Options) *windows.Clipboard {
	return windows.New(logger, opts)
}
