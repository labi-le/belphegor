//go:build windows

package clipboard

import "github.com/labi-le/belphegor/pkg/clipboard/windows"

func New() *windows.Clipboard {
	return windows.New()
}
