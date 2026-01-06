//go:build debug

package lock

import "github.com/rs/zerolog"

func Must(logger zerolog.Logger) func() {
	return func() {}
}
