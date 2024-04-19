package main

import (
	//_ "net/http/pprof"
	"errors"
	"flag"
	"fmt"
	"github.com/labi-le/belphegor/internal"
	"github.com/labi-le/belphegor/internal/node"
	"github.com/nightlyone/lockfile"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
	"time"
)

const LockFile = "belphegor.lck"

var (
	addressIP     string
	port          int
	nodeDiscover  bool
	scanDelay     time.Duration
	discoverDelay time.Duration
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
	flag.DurationVar(&discoverDelay, "discover_delay", 0, "Delay between node discovery")
	flag.DurationVar(&scanDelay, "scan_delay", 0, "Delay between scan local clipboard")
	flag.BoolVar(&debug, "debug", false, "Show debug logs")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&showHelp, "help", false, "Show help")

	flag.Parse()

	initLogger(debug)
}

func main() {
	// if debug {
	//	go http.ListenAndServe("0.0.0.0:8080", nil)
	// }

	if debug {
		log.Info().Msg("debug mode enabled")
	}

	if showVersion {
		_, _ = fmt.Fprintf(os.Stderr,
			"version %s | commit %s | build time %s",
			internal.Version,
			internal.CommitHash,
			internal.BuildTime,
		)
		return
	}

	if showHelp {
		_, _ = fmt.Fprint(os.Stderr, internal.HelpMsg)
		return
	}

	nd := node.New(node.Options{
		Port:               port,
		DiscoverDelay:      discoverDelay,
		ClipboardScanDelay: scanDelay,
	})

	lock := MustLock()
	defer Unlock(lock)

	go func() {
		if err := nd.Start(); err != nil {
			log.Fatal().Err(err).Str("listener", "failed to start the node")
		}
	}()

	if addressIP != "" {
		go func() {
			if err := nd.ConnectTo(addressIP); err != nil {
				log.Fatal().Err(err).Str("connect", "failed to connect to the node")
			}
		}()
	}

	if nodeDiscover {
		go nd.EnableNodeDiscover()
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
