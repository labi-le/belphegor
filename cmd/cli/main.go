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
	"github.com/labi-le/belphegor/internal/service"
	"github.com/labi-le/belphegor/internal/store"
	"github.com/labi-le/belphegor/internal/transport/quic"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/network"
	"github.com/rs/zerolog"
	flag "github.com/spf13/pflag"
)

type config struct {
	addressIP      string
	secret         string
	fileSavePath   string
	maxFileSizeRaw string
	maxFileSize    uint64

	verbose        bool
	showVersion    bool
	showHelp       bool
	notify         bool
	hidden         bool
	installService bool
	discoverEnable bool

	port          int
	maxPeers      int
	discoverDelay time.Duration
	keepAlive     time.Duration
	writeTimeout  time.Duration
	readTimeout   time.Duration
}

func parseFlags() *config {
	cfg := &config{}

	flag.StringVarP(&cfg.addressIP, "connect", "c", "", "Address in ip:port format to connect to the node")
	flag.IntVarP(&cfg.port, "port", "p", netstack.RandomPort(), "Port to use. Default: random")
	flag.BoolVar(&cfg.discoverEnable, "node_discover", true, "Find local nodes on the network and connect to them")
	flag.DurationVar(&cfg.discoverDelay, "discover_delay", 5*time.Minute, "Delay between node discovery")
	flag.DurationVar(&cfg.keepAlive, "keep_alive", 1*time.Minute, "Interval for checking connections between nodes")
	flag.DurationVar(&cfg.writeTimeout, "write_timeout", time.Minute, "Write timeout")
	flag.DurationVar(&cfg.readTimeout, "read_timeout", time.Minute, "Read timeout")
	flag.IntVar(&cfg.maxPeers, "max_peers", 5, "Maximum number of discovered peers")
	flag.BoolVar(&cfg.verbose, "verbose", false, "Verbose logs")
	flag.BoolVar(&cfg.notify, "notify", true, "Enable notifications")
	flag.BoolVarP(&cfg.showVersion, "version", "v", false, "Show version")
	flag.BoolVarP(&cfg.showHelp, "help", "h", false, "Show help")
	flag.BoolVar(&cfg.hidden, "hidden", true, "Hide console window (for windows user)")
	flag.BoolVar(&cfg.installService, "install-service", false, "Install systemd-unit and start the service")
	flag.StringVar(&cfg.secret, "secret", "", "Key to connect between node (empty=all may connect)")
	flag.StringVar(&cfg.maxFileSizeRaw, "max_file_size", "500MiB", "Maximum file size to receive")
	flag.StringVar(&cfg.fileSavePath, "file_save_path", path.Join(os.TempDir(), "bfg_cache"), "Folder where the files sent to us will be saved")

	flag.Parse()

	if cfg.showHelp {
		return cfg
	}

	size, err := humanize.ParseBytes(cfg.maxFileSizeRaw)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "invalid max_file_size format: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}
	cfg.maxFileSize = size

	if cfg.maxPeers <= 0 {
		cfg.maxPeers = 5
	}

	return cfg
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg := parseFlags()

	if cfg.showHelp {
		flag.Usage()
		return
	}

	applyTagsOverrides(cfg)
	logger := initLogger(cfg.verbose)

	logger.Info().
		Str("v", metadata.Version).
		Str("commit_hash", metadata.CommitHash).
		Str("build_time", metadata.BuildTime).
		Send()

	if cfg.showVersion {
		// ^
		return
	}

	if cfg.verbose {
		logger.Info().Msg("verbose mode enabled")
	}

	if cfg.installService {
		if err := service.InstallService(logger); err != nil {
			logger.Fatal().Err(err).Msg("failed install service")
		}
		return
	}

	unlock := lock.Must(logger)
	defer unlock()

	if cfg.hidden {
		logger.Trace().Msg("starting as hidden")
		go console.HideConsoleWindow(cancel)
	}

	nodeOptions := []node.Option{
		node.WithLogger(logger),
		node.WithPublicPort(cfg.port),
		node.WithKeepAlive(cfg.keepAlive),
		node.WithDeadline(network.Deadline{Read: cfg.readTimeout, Write: cfg.writeTimeout}),
		node.WithNotifier(notification.New(cfg.notify)),
		node.WithDiscovering(node.DiscoverOptions{
			Enable:   cfg.discoverEnable,
			Delay:    cfg.discoverDelay,
			MaxPeers: cfg.maxPeers,
		}),
		node.WithSecret(cfg.secret),
		node.WithMaxPeers(cfg.maxPeers),
		node.WithMaxReceiveSize(cfg.maxFileSize),
		node.WithFileStore(store.MustFileStore(cfg.fileSavePath, logger)),
	}

	nodeSettings := node.NewOptions(nodeOptions...)

	tlsConfig, err := security.MakeTLSConfig(cfg.secret, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to generate TLS config")
	}

	quicTransport := quic.New(tlsConfig, nodeSettings.KeepAlive)

	nd := node.New(
		quicTransport,
		clipboard.New(logger),
		new(node.Storage),
		channel.New(cfg.maxPeers),
		nodeOptions...,
	)
	defer func(nd *node.Node) {
		if err := nd.Close(); err != nil {
			logger.Warn().Err(err).Msg("close self node")
		}
	}(nd)

	if cfg.addressIP != "" {
		go func() {
			if err := nd.ConnectTo(ctx, cfg.addressIP); err != nil {
				logger.Fatal().AnErr("node.ConnectTo", err).Msg("failed to connect to the node")
			}
		}()
	}

	if cfg.discoverEnable {
		go discovering.New(
			discovering.WithLogger(logger),
			discovering.WithMaxPeers(cfg.maxPeers),
			discovering.WithDelay(cfg.discoverDelay),
			discovering.WithPort(cfg.port),
		).Discover(ctx, nd)
	}

	if err := nd.Start(ctx); err != nil {
		logger.Fatal().Err(err).Msg("failed to start node")
	}
}

func initLogger(verbose bool) zerolog.Logger {
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}

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
