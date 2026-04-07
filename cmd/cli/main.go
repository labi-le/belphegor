package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/labi-le/belphegor/internal/channel"
	"github.com/labi-le/belphegor/internal/console"
	"github.com/labi-le/belphegor/internal/discovering"
	"github.com/labi-le/belphegor/internal/lock"
	"github.com/labi-le/belphegor/internal/metadata"
	"github.com/labi-le/belphegor/internal/node"
	"github.com/labi-le/belphegor/internal/notification"
	"github.com/labi-le/belphegor/internal/security"
	"github.com/labi-le/belphegor/internal/service"
	"github.com/labi-le/belphegor/internal/store"
	"github.com/labi-le/belphegor/internal/transport"
	"github.com/labi-le/belphegor/internal/transport/quic"
	"github.com/labi-le/belphegor/internal/transport/tcp"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/rs/zerolog"
	flag "github.com/spf13/pflag"
)

func parseFlags() (node.Options, string) {
	var (
		opts      = node.DefaultOptions()
		defaults  = node.DefaultOptions()
		connectTo string
	)

	flag.IntVarP(&opts.ListenPort, "port", "p", defaults.ListenPort, "Listen port")
	flag.BoolVar(&opts.Discovering.Enable, "node_discover", defaults.Discovering.Enable, "Find local nodes on the network and connect to them")
	flag.DurationVar(&opts.Discovering.Delay, "discover_delay", defaults.Discovering.Delay, "Delay between node discovery")
	flag.DurationVar(&opts.KeepAlive, "keep_alive", defaults.KeepAlive, "Interval for checking connections between nodes")
	flag.DurationVar(&opts.Deadline.Write, "write_timeout", defaults.Deadline.Write, "Write timeout")
	flag.DurationVar(&opts.Deadline.Read, "read_timeout", defaults.Deadline.Read, "Read timeout")
	flag.IntVar(&opts.MaxPeers, "max_peers", defaults.MaxPeers, "Maximum number of discovered peers")
	flag.StringVar(&opts.Secret, "secret", defaults.Secret, "Key to connect between node (empty=all may connect)")
	flag.BoolVar(&opts.Clip.AllowCopyFiles, "allow_copy_files", defaults.Clip.AllowCopyFiles, "Allow to copy files")
	flag.IntVar(&opts.Clip.MaxClipboardFiles, "max_clipboard_files", defaults.Clip.MaxClipboardFiles, "Maximum number of files that can be copied (and announced) in a single copy operation")
	flag.Var(&opts.Transport, "transport", "Transport protocol: quic, tcp")

	flag.StringVarP(&connectTo, "connect", "c", "", "Address in ip:port format to connect to the node")
	flag.BoolVar(&opts.Verbose, "verbose", defaults.Verbose, "Verbose logs")
	flag.BoolVar(&opts.Notify, "notify", defaults.Notify, "Enable notifications")
	flag.BoolVarP(&opts.ShowVersion, "version", "v", defaults.ShowVersion, "Show version")
	flag.BoolVarP(&opts.ShowHelp, "help", "h", defaults.ShowHelp, "Show help")
	flag.BoolVar(&opts.Hidden, "hidden", defaults.Hidden, "Hide console window (for windows user)")
	flag.BoolVar(&opts.InstallService, "install_service", defaults.InstallService, "Install systemd-unit and start the service")

	flag.Var(&opts.Clip.MaxFileSize, "max_file_size", "Maximum file size to receive (e.g. 500MiB)")
	flag.StringVar(&opts.FileSavePath, "file_save_path", defaults.FileSavePath, "Folder where the files sent to us will be saved")

	flag.Parse()

	if opts.ShowHelp {
		flag.Usage()
		os.Exit(0)
	}

	return opts.Validated(), connectTo
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts, addressIP := parseFlags()

	applyTagsOverrides(&opts)
	logger := initLogger(opts.Verbose)

	logger.Info().
		Str("v", metadata.Version).
		Str("commit_hash", metadata.CommitHash).
		Str("build_time", metadata.BuildTime).
		Send()

	if opts.ShowVersion {
		// ^
		return
	}

	if opts.Verbose {
		logger.Trace().Object("opts", opts).Msg("verbose mode enabled")
	}

	if !opts.Clip.AllowCopyFiles {
		logger.Warn().Msg("copy file not allowed")
	}

	if opts.InstallService {
		if err := service.InstallService(logger); err != nil {
			logger.Fatal().Err(err).Msg("failed install service")
		}
		return
	}

	unlock := lock.Must(logger)
	defer unlock()

	if opts.Hidden {
		logger.Trace().Msg("starting as hidden")
		go console.HideConsoleWindow(cancel)
	}

	opts.Logger = logger
	opts.Notifier = notification.New(opts.Notify)
	opts.Store = store.MustFileStore(opts.FileSavePath, logger)

	tlsConfig, err := security.MakeTLSConfig(opts.Secret, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to generate TLS config")
	}

	nd := node.New(
		selectTransport(opts.Transport, tlsConfig, opts.KeepAlive, logger),
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

	if addressIP != "" {
		go func() {
			if err := nd.ConnectTo(ctx, addressIP); err != nil {
				logger.Fatal().AnErr("node.ConnectTo", err).Msg("failed to connect to the node")
			}
		}()
	}

	if opts.Discovering.Enable {
		go discovering.New(
			discovering.WithLogger(logger),
			discovering.WithMaxPeers(opts.MaxPeers),
			discovering.WithDelay(opts.Discovering.Delay),
			discovering.WithPort(opts.ListenPort),
		).Discover(ctx, nd)
	}

	if err := nd.Start(ctx); err != nil {
		logger.Fatal().Err(err).Msg("failed to start node")
	}
}

func selectTransport(
	mode node.Transport,
	tlsConf *tls.Config,
	keepAlive time.Duration,
	logger zerolog.Logger,
) transport.Transport {
	ctxLog := logger.With().Str("op", "selectTransport").Logger()
	if mode == node.TransportTCP {
		ctxLog.Info().Msg("selected tcp")
		return tcp.New(tlsConf, keepAlive)
	}

	ctxLog.Info().Msg("selected quic")
	return quic.New(tlsConf, keepAlive)
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
