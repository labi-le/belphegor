//go:build debug

package main

import (
	"net/http"
	_ "net/http/pprof"
)

func applyTagsOverrides(cfg *action) {
	cfg.verbose = true
	cfg.notify = false

	go func() {
		addr := "0.0.0.0:6060"
		if err := http.ListenAndServe(addr, nil); err != nil {
			panic(err)
		}
	}()
}
