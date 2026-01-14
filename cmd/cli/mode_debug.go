//go:build debug

package main

import (
	"net/http"
	_ "net/http/pprof"

	"github.com/rs/zerolog"
)

func applyTagsOverrides(cfg *config, logger zerolog.Logger) {
	cfg.verbose = true
	cfg.port = 7777
	cfg.notify = false

	go func() {
		addr := "0.0.0.0:6060"
		logger.Debug().Msgf("starting pprof server on %s", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			logger.Fatal().Msgf("pprof server failed: %v", err)
		}
	}()
}
