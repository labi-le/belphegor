package ctxlog

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Op(op string) zerolog.Logger {
	return log.With().Str("op", op).Logger()
}
