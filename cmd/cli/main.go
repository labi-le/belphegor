package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/labi-le/belphegor/internal"
	"github.com/labi-le/belphegor/internal/discovering"
	"github.com/labi-le/belphegor/internal/netstack"
	"github.com/labi-le/belphegor/internal/node"
	"github.com/labi-le/belphegor/internal/notification"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/console"
	"github.com/labi-le/belphegor/pkg/id"
	"github.com/labi-le/belphegor/pkg/storage"
	"github.com/nightlyone/lockfile"
	"github.com/rs/zerolog"
	flag "github.com/spf13/pflag"
)

const LockFile = "belphegor.lck"

var (
	addressIP   string
	debug       bool
	showVersion bool
	showHelp    bool
	notify      bool
	hidden      bool

	port           int
	discoverEnable bool
	discoverDelay  time.Duration
	scanDelay      time.Duration
	keepAlive      time.Duration
	writeTimeout   time.Duration
	maxPeers       int
	bitSize        int
)

var (
	ErrCannotLock     = errors.New("cannot get locked process: %s")
	ErrCannotUnlock   = errors.New("cannot unlock process: %s")
	ErrAlreadyRunning = errors.New("belphegor is already running. pid %d")
)

func init() {
	flag.StringVarP(&addressIP, "connect", "c", "", "Address in ip:port format to connect to the node")
	flag.IntVarP(&port, "port", "p", netstack.RandomPort(), "Port to use. Default: random")
	flag.BoolVar(&discoverEnable, "node_discover", true, "Find local nodes on the network and connect to them")
	flag.DurationVar(&discoverDelay, "discover_delay", 5*time.Minute, "Delay between node discovery")
	flag.DurationVar(&scanDelay, "scan_delay", 2*time.Second, "Delay between scan local clipboard")
	flag.DurationVar(&keepAlive, "keep_alive", 1*time.Minute, "Interval for checking connections between nodes")
	flag.DurationVar(&writeTimeout, "write_timeout", 5*time.Second, "Write timeout")
	flag.IntVar(&maxPeers, "max_peers", 5, "Maximum number of peers to connect to")
	flag.IntVar(&bitSize, "bit_size", 2048, "RSA key bit size")
	flag.BoolVarP(&debug, "debug", "d", false, "Show debug logs")
	flag.BoolVar(&notify, "notify", true, "Enable notifications")
	flag.BoolVarP(&showVersion, "version", "v", false, "Show version")
	flag.BoolVarP(&showHelp, "help", "h", false, "Show help")
	flag.BoolVar(&hidden, "hidden", false, "Hide console window (for windows user)")

	flag.Parse()
}

func main() {
	logger := initLogger(debug)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if debug {
		port = 7777
		logger.Info().Msg("debug mode enabled")
	}

	if showVersion {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"version %s | commit %s | build time %s",
			internal.Version,
			internal.CommitHash,
			internal.BuildTime,
		)
		return
	}

	if showHelp {
		flag.Usage()
		return
	}

	if hidden {
		console.HideConsoleWindow()
	}

	nd := node.New(
		clipboard.New(logger),
		storage.NewSyncMapStorage[id.Unique, *node.Peer](),
		node.NewChannel(),
		node.WithLogger(logger),
		node.WithPublicPort(port),
		node.WithBitSize(bitSize),
		node.WithKeepAlive(keepAlive),
		node.WithClipboardScanDelay(scanDelay),
		node.WithWriteTimeout(writeTimeout),
		node.WithNotifier(notificationProvider(notify)),
		node.WithDiscovering(node.DiscoverOptions{
			Enable:   discoverEnable,
			Delay:    discoverDelay,
			MaxPeers: maxPeers,
		}),
	)

	lock := MustLock(logger)
	defer Unlock(lock, logger)

	if addressIP != "" {
		go func() {
			if err := nd.ConnectTo(ctx, addressIP); err != nil {
				logger.Fatal().AnErr("node.ConnectTo", err).Msg("failed to connect to the node")
			}
		}()
	}

	if discoverEnable {
		go discovering.New(
			discovering.WithLogger(logger),
			discovering.WithMaxPeers(maxPeers),
			discovering.WithDelay(discoverDelay),
			discovering.WithPort(port),
		).Discover(ctx, nd)
	}

	if err := nd.Start(ctx); err != nil {
		logger.Fatal().Err(err).Msg("failed to start node")
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

func initLogger(debug bool) zerolog.Logger {
	output := zerolog.ConsoleWriter{Out: os.Stderr}

	if debug {
		zerolog.CallerMarshalFunc = func(_ uintptr, file string, line int) string {
			short := file
			for i := len(file) - 1; i > 0; i-- {
				if file[i] == '/' {
					short = file[i+1:]
					break
				}
			}
			file = short
			return fmt.Sprintf("%s:%d", file, line)
		}
		return zerolog.New(output).
			Level(zerolog.TraceLevel).
			With().
			Timestamp().
			Caller().
			Logger()
	}

	return zerolog.New(output).
		Level(zerolog.InfoLevel).
		With().
		Timestamp().
		Logger()
}

func MustLock(l zerolog.Logger) lockfile.Lockfile {
	lock, _ := lockfile.New(filepath.Join(os.TempDir(), LockFile))

	if lockErr := lock.TryLock(); lockErr != nil {
		owner, err := lock.GetOwner()
		if err != nil {
			l.Fatal().Msgf(ErrCannotLock.Error(), err)
		}
		l.Fatal().Msgf(ErrAlreadyRunning.Error(), owner.Pid)
	}

	return lock
}

func Unlock(lock lockfile.Lockfile, l zerolog.Logger) {
	if err := lock.Unlock(); err != nil {
		l.Fatal().Msgf(ErrCannotUnlock.Error(), err)
	}
}
