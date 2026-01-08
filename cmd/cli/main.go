package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/labi-le/belphegor/internal/channel"
	"github.com/labi-le/belphegor/internal/console"
	"github.com/labi-le/belphegor/internal/discovering"
	"github.com/labi-le/belphegor/internal/lock"
	"github.com/labi-le/belphegor/internal/metadata"
	"github.com/labi-le/belphegor/internal/netstack"
	"github.com/labi-le/belphegor/internal/node"
	"github.com/labi-le/belphegor/internal/notification"
	"github.com/labi-le/belphegor/internal/security"
	"github.com/labi-le/belphegor/internal/store"
	"github.com/labi-le/belphegor/internal/transport/quic"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/network"
	"github.com/rs/zerolog"
	flag "github.com/spf13/pflag"
)

var (
	options []node.Option
	logger  zerolog.Logger
)

var (
	addressIP string
	secret    string

	fileSavePath string

	maxFileSizeRaw string
	maxFileSize    uint64

	verbose     bool
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
	flag.BoolVar(&verbose, "verbose", false, "Verbose logs")
	flag.BoolVar(&notify, "notify", true, "Enable notifications")
	flag.BoolVarP(&showVersion, "version", "v", false, "Show version")
	flag.BoolVarP(&showHelp, "help", "h", false, "Show help")
	flag.BoolVar(&hidden, "hidden", true, "Hide console window (for windows user)")
	flag.StringVar(&secret, "secret", "", "Key to connect between node (empty=all may connect)")
	flag.StringVar(&maxFileSizeRaw, "max_file_size", "500MiB", "Maximum number of discovered peers")
	flag.StringVar(&fileSavePath, "file_save_path", path.Join(os.TempDir(), "bfg_cache"), "Folder where the files sent to us will be saved")

	flag.Parse()

	size, err := humanize.ParseBytes(maxFileSizeRaw)
	if err != nil {
		flag.Usage()
		return
	}
	maxFileSize = size

	logger = initLogger(verbose)
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if verbose {
		logger.Info().Msg("verbose mode enabled")
	}

	if showHelp {
		flag.Usage()
		return
	}

	if showVersion {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"version: %s\ncommit: %s\nbuild time: %s\n",
			metadata.Version,
			metadata.CommitHash,
			metadata.BuildTime,
		)
		return
	}

	if hidden {
		logger.Trace().Msg("starting as hidden")
		go console.HideConsoleWindow(cancel)
	}

	options = append([]node.Option{
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
		node.WithSecret(secret),
		node.WithMaxPeers(maxPeers),
		node.WithMaxReceiveSize(maxFileSize),
		node.WithFileStore(store.MustFileStore(fileSavePath, logger)),
	}, options...)

	nodeSettings := node.NewOptions(options...)

	tlsConfig, err := security.MakeTLSConfig(secret, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to generate TLS config")
	}

	quicTransport := quic.New(tlsConfig, nodeSettings.KeepAlive)

	nd := node.New(
		quicTransport,
		clipboard.New(logger),
		new(node.Storage),
		channel.New(),
		options...,
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

func initLogger(verbose bool) zerolog.Logger {
	output := zerolog.ConsoleWriter{Out: os.Stderr}

	if verbose {
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
