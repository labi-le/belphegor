//go:build debug

package main

import (
	"net/http"
	_ "net/http/pprof"
)

func init() {
	go func() {
		addr := "0.0.0.0:6060"
		logger.Debug().Msgf("starting pprof server on %s", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			logger.Debug().Msgf("pprof server failed: %v", err)
		}
	}()
	//options = append(options, node.WithPublicPort(7777))
}
