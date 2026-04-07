//go:build debug

package main

import (
	"net/http"
	_ "net/http/pprof"

	"github.com/labi-le/belphegor/internal/node"
)

func applyTagsOverrides(opts *node.Options) {
	opts.Verbose = true
	opts.Notify = false

	go func() {
		addr := "0.0.0.0:6060"
		if err := http.ListenAndServe(addr, nil); err != nil {
			panic(err)
		}
	}()
}
