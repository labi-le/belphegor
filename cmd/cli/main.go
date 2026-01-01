package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/labi-le/belphegor/internal/channel"
	"github.com/labi-le/belphegor/internal/discovering"
	"github.com/labi-le/belphegor/internal/lock"
	"github.com/labi-le/belphegor/internal/metadata"
	"github.com/labi-le/belphegor/internal/netstack"
	"github.com/labi-le/belphegor/internal/node"
	"github.com/labi-le/belphegor/internal/notification"
	"github.com/labi-le/belphegor/internal/peer"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/console"
	"github.com/labi-le/belphegor/pkg/id"
	"github.com/labi-le/belphegor/pkg/network"
	"github.com/labi-le/belphegor/pkg/storage"
	"github.com/rs/zerolog"
	flag "github.com/spf13/pflag"
)

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
	keepAlive      time.Duration
	writeTimeout   time.Duration
	readTimeout    time.Duration
	maxPeers       int
)

func init() {
	flag.StringVarP(&addressIP, "connect", "c", "", "Address in ip:port format to connect to the node")
	flag.IntVarP(&port, "port", "p", netstack.RandomPort(), "Port to use. Default: random")
	flag.BoolVar(&discoverEnable, "node_discover", true, "Find local nodes on the network and connect to them")
	flag.DurationVar(&discoverDelay, "discover_delay", 5*time.Minute, "Delay between node discovery")
	flag.DurationVar(&keepAlive, "keep_alive", 1*time.Minute, "Interval for checking connections between nodes")
	flag.DurationVar(&writeTimeout, "write_timeout", time.Minute, "Write timeout")
	flag.DurationVar(&readTimeout, "read_timeout", time.Minute, "Write timeout")
	flag.IntVar(&maxPeers, "max_peers", 5, "Maximum number of discovered peers")
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
			metadata.Version,
			metadata.CommitHash,
			metadata.BuildTime,
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
		storage.NewSyncMapStorage[id.Unique, *peer.Peer](),
		channel.NewChannel(),
		node.WithLogger(logger),
		node.WithPublicPort(port),
		node.WithKeepAlive(keepAlive),
		node.WithDeadline(network.Deadline{
			Read:  readTimeout,
			Write: writeTimeout,
		}),
		node.WithNotifier(notification.New(notify)),
		node.WithDiscovering(node.DiscoverOptions{
			Enable:   discoverEnable,
			Delay:    discoverDelay,
			MaxPeers: maxPeers,
		}),
	)

	unlock := lock.Must(logger)
	defer unlock()

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
