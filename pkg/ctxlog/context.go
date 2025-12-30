package ctxlog

import (
	"github.com/rs/zerolog"
)

func Op(logger zerolog.Logger, op string) zerolog.Logger {
	return logger.With().Str("op", op).Logger()
}
