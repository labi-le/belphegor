//go:build windows

package clipboard

import (
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/clipboard/windows"
	"github.com/rs/zerolog"
)

func New(opts eventful.Options, logger zerolog.Logger) *windows.Clipboard {
	return windows.New(opts, logger)
}
