//go:build !debug

package security

import (
	"crypto/tls"

	"github.com/rs/zerolog"
)

func populateKeyLog(_ zerolog.Logger, _ *tls.Config) {}
