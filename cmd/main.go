package main

import (
	"belphegor/internal/belphegor"
	"belphegor/pkg/clipboard"
	"flag"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"time"
)

var version = "dev"

var (
	helpMsg = `belphegor - 
A cross-platform clipboard sharing utility

Usage:
	belphegor [flags]

Flags:
	-connect string | ip:port to connect to the node (e.g. 192.168.0.12:7777)
	-port int | the node will start on this port (e.g. 7777)
    -node_discover bool | find local nodes on the network and connect to them
	-scan_delay string | delay between scan local clipboard (e.g. 5s)
	-debug | show debug logs
	-version | show version
	-help | show help
`
	addressIP     string
	port          int
	nodeDiscover  bool
	scanDelay     string
	discoverDelay string
	debug         bool
	showVersion   bool
	showHelp      bool
)

func init() {
	flag.StringVar(&addressIP, "connect", "", "Address in ip:port format to connect to the node")
	flag.IntVar(&port, "port", 0, "Port to use. Default: random")
	flag.BoolVar(&nodeDiscover, "node_discover", true, "Find local nodes on the network and connect to them")
	flag.StringVar(&discoverDelay, "discover_delay", "60s", "Delay between node discovery")
	flag.StringVar(&scanDelay, "scan_delay", "5s", "Delay between scan local clipboard")
	flag.BoolVar(&debug, "debug", false, "Show debug logs")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&showHelp, "help", false, "Show help")

	flag.Parse()

	initLogger(debug)
}

func main() {
	if debug {
		log.Info().Msg("debug mode enabled")
	}

	if showVersion {
		log.Printf("version %s", version)
	}

	if showHelp {
		_, _ = fmt.Fprint(os.Stderr, helpMsg)
		os.Exit(0)
	}

	discoverDelayDuration, delayErr := time.ParseDuration(discoverDelay)
	if delayErr != nil {
		log.Fatal().Err(delayErr).Msg("failed to parse discover delay")
	}

	scanDelayDuration, scanDelayErr := time.ParseDuration(scanDelay)
	if scanDelayErr != nil {
		log.Fatal().Err(delayErr).Msg("failed to parse scan delay")
	}

	var (
		node    *belphegor.Node
		storage = belphegor.NewSyncMapStorage()
		cp      = clipboard.NewThreadSafe()
	)
	if port != 0 {
		node = belphegor.NewNode(
			cp,
			port,
			discoverDelayDuration,
			storage,
			belphegor.NewChannel(),
		)
	} else {
		log.Debug().Msg("using random port")
		node = belphegor.NewNodeRandomPort(
			cp,
			discoverDelayDuration,
			storage,
			belphegor.NewChannel(),
		)
	}

	go func() {
		if err := node.Start(scanDelayDuration); err != nil {
			log.Fatal().Err(err).Msg("failed to start the node")
		}
	}()

	if addressIP != "" {
		go func() {
			if err := node.ConnectTo(addressIP); err != nil {
				log.Fatal().Err(err).Msg("failed to connect to the node")
			}
		}()
	}

	if nodeDiscover {
		log.Debug().Msg("node discovery enabled, delay: " + discoverDelay)
		go node.EnableNodeDiscover()
	}

	select {}
}

func initLogger(debug bool) {
	if debug {
		log.Logger = log.With().Caller().Logger()
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
		return
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}
