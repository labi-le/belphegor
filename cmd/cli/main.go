package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/labi-le/belphegor/internal"
	"github.com/labi-le/belphegor/internal/netstack"
	"github.com/labi-le/belphegor/internal/node"
	"github.com/labi-le/belphegor/internal/node/data"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/storage"
	"github.com/nightlyone/lockfile"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

const LockFile = "belphegor.lck"

var (
	addressIP   string
	debug       bool
	showVersion bool
	showHelp    bool
	port        int

	options = &node.Options{
		Metadata: data.SelfMetaData(),
	}
)

var (
	ErrCannotLock     = errors.New("cannot get locked process: %s")
	ErrCannotUnlock   = errors.New("cannot unlock process: %s")
	ErrAlreadyRunning = errors.New("belphegor is already running. pid %d")
)

func init() {
	flag.IntVar(&port, "port", netstack.RandomPort(), "Port to use. Default: random")

	flag.StringVar(&addressIP, "connect", "", "Address in ip:port format to connect to the node")

	flag.BoolVar(&options.Discovering.Enable, "node_discover", true, "Find local nodes on the network and connect to them")
	flag.DurationVar(&options.Discovering.SearchDelay, "discover_delay", 5*time.Minute, "Delay between node discovery")
	flag.IntVar(&options.Discovering.MaxPeers, "max_peers", 5, "Maximum number of peers to connect to")

	flag.DurationVar(&options.ClipboardScanDelay, "scan_delay", 2*time.Second, "Delay between scan local clipboard")
	flag.DurationVar(&options.KeepAlive, "keep_alive", 1*time.Minute, "Interval for checking connections between nodes")
	flag.DurationVar(&options.WriteTimeout, "write_timeout", 5*time.Second, "Write timeout")

	flag.BoolVar(&options.Encryption.Enable, "enable_enc", false, "Enable encryption")

	flag.BoolVar(&debug, "debug", false, "Show debug logs")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&showHelp, "help", false, "Show help")

	flag.Parse()

	options.PublicPort = port
	options.Discovering.Port = port

	initLogger(debug)
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

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

	nd := node.New(
		clipboard.NewThreadSafe(),
		storage.NewSyncMapStorage[data.UniqueID, *node.Peer](),
		make(data.Channel),
		options,
	)

	lock := MustLock()
	defer Unlock(lock)

	if addressIP != "" {
		go func() {
			if err := nd.Connect(ctx, addressIP); err != nil {
				log.Fatal().AnErr("node.Connect", err).Msg("failed to connect to the node")
			}
		}()
	}

	if err := nd.Start(ctx); err != nil {
		log.Fatal().AnErr("node.Start", err).Msg("failed to start the node")
	}
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
