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
