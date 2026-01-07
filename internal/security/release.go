//go:build !debug

package security

import (
	"crypto/tls"

	"github.com/rs/zerolog"
)

// populateKeyLog is a no-op stub for release builds.
func populateKeyLog(_ zerolog.Logger, _ *tls.Config) {}
