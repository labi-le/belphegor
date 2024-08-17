package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/labi-le/belphegor/internal"
	"github.com/labi-le/belphegor/internal/discovering"
	"github.com/labi-le/belphegor/internal/netstack"
	"github.com/labi-le/belphegor/internal/node"
	"github.com/labi-le/belphegor/internal/node/data"
	"github.com/labi-le/belphegor/internal/notification"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/storage"
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
	keepAlive     time.Duration
	writeTimeout  time.Duration
	maxPeers      int
	bitSize       int
	debug         bool
	showVersion   bool
	showHelp      bool
	notify        bool
)

var (
	ErrCannotLock     = errors.New("cannot get locked process: %s")
	ErrCannotUnlock   = errors.New("cannot unlock process: %s")
	ErrAlreadyRunning = errors.New("belphegor is already running. pid %d")
)

func init() {
	flag.StringVar(&addressIP, "connect", "", "Address in ip:port format to connect to the node")
	flag.IntVar(&port, "port", netstack.RandomPort(), "Port to use. Default: random")
	flag.BoolVar(&nodeDiscover, "node_discover", true, "Find local nodes on the network and connect to them")
	flag.DurationVar(&discoverDelay, "discover_delay", 5*time.Minute, "Delay between node discovery")
	flag.DurationVar(&scanDelay, "scan_delay", 2*time.Second, "Delay between scan local clipboard")
	flag.DurationVar(&keepAlive, "keep_alive", 1*time.Minute, "Interval for checking connections between nodes")
	flag.DurationVar(&writeTimeout, "write_timeout", 5*time.Second, "Write timeout")
	flag.IntVar(&maxPeers, "max_peers", 5, "Maximum number of peers to connect to")
	flag.IntVar(&bitSize, "bit_size", 2048, "RSA key bit size")
	flag.BoolVar(&debug, "debug", false, "Show debug logs")
	flag.BoolVar(&notify, "notify", true, "Enable notifications")
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

	opts := &node.Options{
		PublicPort:         uint16(port),
		BitSize:            uint16(bitSize),
		KeepAlive:          keepAlive,
		ClipboardScanDelay: scanDelay,
		WriteTimeout:       writeTimeout,
		Metadata:           data.SelfMetaData(),
		Notifier:           notificationProvider(notify),
	}

	nd := node.New(
		clipboard.NewThreadSafe(),
		storage.NewSyncMapStorage[data.UniqueID, *node.Peer](),
		make(data.Channel),
		opts,
	)

	lock := MustLock()
	defer Unlock(lock)

	if addressIP != "" {
		go func() {
			if err := nd.ConnectTo(addressIP); err != nil {
				log.Fatal().AnErr("node.ConnectTo", err).Msg("failed to connect to the node")
			}
		}()
	}

	if nodeDiscover {
		go discovering.New(
			maxPeers,
			discoverDelay,
			port,
		).Discover(nd)
	}

	if err := nd.Start(); err != nil {
		log.Fatal().AnErr("node.Start", err).Msg("failed to start the node")
	}
}

func notificationProvider(enable bool) notification.Notifier {
	if enable {
		return notification.BeepDecorator{
			Title: "Belphegor",
		}
	}

	return new(notification.NullNotifier)
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
