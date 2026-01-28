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
	"github.com/rs/zerolog"
	flag "github.com/spf13/pflag"
)

type action struct {
	addressIP string

	verbose        bool
	showVersion    bool
	showHelp       bool
	notify         bool
	hidden         bool
	installService bool

	fileSavePath string
}

func parseFlags() (node.Options, action) {
	var (
		opts = node.DefaultOptions
		act  action
	)

	flag.IntVarP(&opts.PublicPort, "port", "p", netstack.RandomPort(), "Port to use. Default: random")
	flag.BoolVar(&opts.Discovering.Enable, "node_discover", true, "Find local nodes on the network and connect to them")
	flag.DurationVar(&opts.Discovering.Delay, "discover_delay", 30*time.Second, "Delay between node discovery")
	flag.DurationVar(&opts.KeepAlive, "keep_alive", 1*time.Minute, "Interval for checking connections between nodes")
	flag.DurationVar(&opts.Deadline.Write, "write_timeout", time.Minute, "Write timeout")
	flag.DurationVar(&opts.Deadline.Read, "read_timeout", time.Minute, "Read timeout")
	flag.IntVar(&opts.MaxPeers, "max_peers", 5, "Maximum number of discovered peers")
	flag.StringVar(&opts.Secret, "secret", "", "Key to connect between node (empty=all may connect)")
	flag.BoolVar(&opts.Clip.AllowCopyFiles, "allow_copy_files", true, "Allow to copy files")
	flag.IntVar(&opts.Clip.MaxClipboardFiles, "max_clipboard_files", 15, "Maximum number of files that can be copied (and announced) in a single copy operation")

	flag.StringVarP(&act.addressIP, "connect", "c", "", "Address in ip:port format to connect to the node")
	flag.BoolVar(&act.verbose, "verbose", false, "Verbose logs")
	flag.BoolVar(&act.notify, "notify", true, "Enable notifications")
	flag.BoolVarP(&act.showVersion, "version", "v", false, "Show version")
	flag.BoolVarP(&act.showHelp, "help", "h", false, "Show help")
	flag.BoolVar(&act.hidden, "hidden", true, "Hide console window (for windows user)")
	flag.BoolVar(&act.installService, "install-service", false, "Install systemd-unit and start the service")

	var maxFileSizeRaw string
	flag.StringVar(&maxFileSizeRaw, "max_file_size", "500MiB", "Maximum file size to receive")
	flag.StringVar(&act.fileSavePath, "file_save_path", path.Join(os.TempDir(), "bfg_cache"), "Folder where the files sent to us will be saved")

	flag.Parse()

	if act.showHelp {
		return opts, act
	}

	size, err := humanize.ParseBytes(maxFileSizeRaw)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "invalid max_file_size format: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}
	opts.Clip.MaxFileSize = size

	if opts.MaxPeers <= 0 {
		opts.MaxPeers = 5
	}

	if opts.Clip.MaxClipboardFiles == 0 {
		opts.Clip.MaxClipboardFiles = 15
	}

	return opts, act
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts, cfg := parseFlags()

	if cfg.showHelp {
		flag.Usage()
		return
	}

	applyTagsOverrides(&cfg)
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

	if !opts.Clip.AllowCopyFiles {
		logger.Warn().Msg("copy file not allowed")
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

	opts.Logger = logger
	opts.Notifier = notification.New(cfg.notify)
	opts.Store = store.MustFileStore(cfg.fileSavePath, logger)

	tlsConfig, err := security.MakeTLSConfig(opts.Secret, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to generate TLS config")
	}

	nd := node.New(
		quic.New(tlsConfig, opts.KeepAlive),
		clipboard.New(opts.Clip, logger),
		new(node.Storage),
		channel.New(opts.MaxPeers),
		opts,
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

	if opts.Discovering.Enable {
		go discovering.New(
			discovering.WithLogger(logger),
			discovering.WithMaxPeers(opts.MaxPeers),
			discovering.WithDelay(opts.Discovering.Delay),
			discovering.WithPort(opts.PublicPort),
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
