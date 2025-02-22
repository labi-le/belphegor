//go:build darwin

package clipboard

import (
	"github.com/labi-le/belphegor/pkg/clipboard/mac"
)

func New() *mac.Darwin {
	return mac.New()
}
