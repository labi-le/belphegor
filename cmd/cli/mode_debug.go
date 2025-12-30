package main

import (
	"net/http"
	_ "net/http/pprof"

	"github.com/rs/zerolog/log"
)

func init() {
	addr := "127.0.0.1:6060"
	log.Logger.Debug().Msgf("pprof started on %s", addr)

	go http.ListenAndServe(addr, nil)
}
