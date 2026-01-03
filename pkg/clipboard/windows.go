//go:build windows

package clipboard

import (
	"github.com/labi-le/belphegor/pkg/clipboard/windows"
	"github.com/rs/zerolog"
)

func New(zerolog.Logger) *windows.Clipboard {
	return windows.New()
}
