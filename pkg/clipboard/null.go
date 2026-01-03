//go:build null

package clipboard

import (
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/clipboard/null"
	"github.com/rs/zerolog"
)

func New(zerolog.Logger) eventful.Eventful {
	return null.NewNull()
}
