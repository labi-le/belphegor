//go:build debug

package main

import (
	"net/http"
	_ "net/http/pprof"
)

func applyTagsOverrides(cfg *config) {
	cfg.verbose = true
	cfg.port = 7777
	cfg.notify = false

	go func() {
		addr := "0.0.0.0:6060"
		if err := http.ListenAndServe(addr, nil); err != nil {
			panic(err)
		}
	}()
}
