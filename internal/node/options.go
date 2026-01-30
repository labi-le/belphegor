package node

import (
	"time"

	"github.com/labi-le/belphegor/internal/netstack"
	"github.com/labi-le/belphegor/internal/notification"
	"github.com/labi-le/belphegor/internal/store"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/network"
	"github.com/rs/zerolog"
)

type Options struct {
	PublicPort  int
	KeepAlive   time.Duration
	Deadline    network.Deadline
	Notifier    notification.Notifier
	Discovering DiscoverOptions
	Metadata    domain.Device
	Logger      zerolog.Logger
	Secret      string
	MaxPeers    int
	Store       store.FileWriter
	Clip        eventful.Options
}

func (o Options) MarshalZerologObject(e *zerolog.Event) {
	e.Int("public_port", o.PublicPort)
	e.Str("keep_alive", o.KeepAlive.String())
	e.Dict(
		"deadline",
		zerolog.Dict().
			Str("read", o.Deadline.Read.String()).
			Str("write", o.Deadline.Write.String()),
	)
	e.Dict(
		"metadata",
		zerolog.Dict().
			Str("arch", o.Metadata.Arch).
			Int64("id", o.Metadata.ID.Int64()).
			Str("name", o.Metadata.Name),
	)
	e.Dict(
		"discovering",
		zerolog.Dict().
			Bool("enable", o.Discovering.Enable).
			Str("delay", o.Discovering.Delay.String()).
			Int("max_peers", o.Discovering.MaxPeers),
	)
	e.Bool("has_secret", o.Secret != "")
	e.Int("max_peers", o.MaxPeers)
	e.Dict(
		"clipboard_options",
		zerolog.Dict().
			Bool("allow_copy_files", o.Clip.AllowCopyFiles).
			Int("max_clipboard_files", o.Clip.MaxClipboardFiles).
			Int64("max_file_size", int64(o.Clip.MaxFileSize)),
	)
}

type DiscoverOptions struct {
	Enable   bool
	Delay    time.Duration
	MaxPeers int
}

type Option func(*Options)

//nolint:mnd //shut up
var DefaultOptions = Options{
	PublicPort: netstack.RandomPort(),
	KeepAlive:  time.Minute,
	Deadline: network.Deadline{
		Read:  5 * time.Second,
		Write: 5 * time.Second,
	},
	Notifier: new(notification.BeepDecorator),
	Discovering: DiscoverOptions{
		Enable:   true,
		Delay:    30 * time.Second,
		MaxPeers: 5,
	},
	Metadata: domain.SelfMetaData(),
	MaxPeers: 4,
}

func NewOptions(opts ...Option) Options {
	options := DefaultOptions

	for _, opt := range opts {
		opt(&options)
	}

	return options
}

func WithPublicPort(port int) Option {
	return func(o *Options) {
		o.PublicPort = port
	}
}

func WithNotifier(notifier notification.Notifier) Option {
	return func(o *Options) {
		o.Notifier = notifier
	}
}

func WithMetadata(opt domain.Device) Option {
	return func(o *Options) {
		o.Metadata = opt
	}
}

func WithLogger(logger zerolog.Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

func WithFileStore(store store.FileWriter) Option {
	return func(options *Options) {
		options.Store = store
	}
}
