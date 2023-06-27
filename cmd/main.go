package main

import (
	"belphegor/internal/belphegor"
	"belphegor/pkg/clipboard"
	"belphegor/pkg/ip"
	"flag"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
)

var version = "dev"

var (
	//todo: add -findLocal flag to find local nodes on the network and connect to them
	helpMsg = `belphegor - 
A cross-platform clipboard sharing utility

Usage:
	belphegor [flags]

Flags:
	-connect string | ip:port to connect to the node (e.g. 192.168.0.12:7777)
	-port int | the node will start on this port (e.g. 7777)
	-debug | show debug logs
	-version | show version
	-help | show help
`
	addressIP   string
	port        int
	debug       bool
	showVersion bool
	showHelp    bool
)

func init() {
	flag.StringVar(&addressIP, "connect", "", "Address in ip:port format to connect to the node")
	flag.IntVar(&port, "port", 0, "Port to use. Default: random")
	flag.BoolVar(&debug, "debug", false, "Show debug logs")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&showHelp, "help", false, "Show help")

	flag.Parse()

	initLogger(debug)
}

func main() {

	if debug {
		log.Info().Msg("Debug mode enabled")
	}

	if showVersion {
		log.Printf("version %s", version)
	}

	if showHelp {
		log.Info().Msg(helpMsg)
		os.Exit(0)
	}

	var node *belphegor.Node
	if port != 0 {
		node = belphegor.NewNode(clipboard.NewManager(), ip.MakeAddr(port))
	} else {
		log.Debug().Msg("Using random port")
		node = belphegor.NewNodeRandomPort(clipboard.NewManager())
	}

	go func() {
		if err := node.Start(); err != nil {
			log.Fatal().Err(err).Msg("Failed to start the node")
		}
	}()

	//todo add neighbor discovery by key or short word
	if addressIP != "" {
		go func() {
			if err := node.ConnectTo(addressIP); err != nil {
				log.Fatal().Err(err).Msg("Failed to connect to the node")
			}
		}()
	}

	select {}
}

func initLogger(debug bool) {
	if debug {
		log.Logger = log.With().Caller().Logger()
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		return
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}
