package main

import (
	//_ "net/http/pprof"
	"belphegor/internal/belphegor"
	"belphegor/pkg/clipboard"
	"errors"
	"flag"
	"fmt"
	"github.com/nightlyone/lockfile"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
	"time"
)

var version = "dev"

const LockFile = "belphegor.lck"

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

var (
	ErrCannotLock     = errors.New("cannot get locked process: %s")
	ErrCannotUnlock   = errors.New("cannot unlock process: %s")
	ErrAlreadyRunning = errors.New("belphegor is already running. pid %d")
)

func init() {
	flag.StringVar(&addressIP, "connect", "", "Address in ip:port format to connect to the node")
	flag.IntVar(&port, "port", 0, "Port to use. Default: random")
	flag.BoolVar(&nodeDiscover, "node_discover", true, "Find local nodes on the network and connect to them")
	flag.StringVar(&discoverDelay, "discover_delay", "60s", "Delay between node discovery")
	flag.StringVar(&scanDelay, "scan_delay", "1s", "Delay between scan local clipboard")
	flag.BoolVar(&debug, "debug", false, "Show debug logs")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&showHelp, "help", false, "Show help")

	flag.Parse()

	initLogger(debug)
}

func main() {
	//if debug {
	//	go http.ListenAndServe("0.0.0.0:8080", nil)
	//}
	lock := MustLock()
	defer Unlock(lock)

	if debug {
		log.Info().Msg("debug mode enabled")
	}

	if showVersion {
		log.Printf("version %s", version)
		return
	}

	if showHelp {
		_, _ = fmt.Fprint(os.Stderr, helpMsg)
		return
	}

	discoverDelayDuration, delayErr := time.ParseDuration(discoverDelay)
	if delayErr != nil {
		log.Panic().Err(delayErr).Msg("failed to parse discover delay")
	}

	scanDelayDuration, scanDelayErr := time.ParseDuration(scanDelay)
	if scanDelayErr != nil {
		log.Panic().Err(delayErr).Msg("failed to parse scan delay")
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
			log.Panic().Err(err).Msg("failed to start the node")
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

func MustLock() lockfile.Lockfile {
	lock, _ := lockfile.New(filepath.Join(os.TempDir(), LockFile))

	if lockErr := lock.TryLock(); lockErr != nil {
		owner, err := lock.GetOwner()
		if err != nil {
			log.Fatal().Msgf(ErrCannotLock.Error(), err)
		}
		log.Fatal().Msgf(ErrAlreadyRunning.Error(), owner.Pid)
	}

	return lock
}

func Unlock(lock lockfile.Lockfile) {
	if err := lock.Unlock(); err != nil {
		log.Fatal().Msgf(ErrCannotUnlock.Error(), err)
	}
}
