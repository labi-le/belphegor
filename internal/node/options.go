package node

import (
	"time"

	"github.com/labi-le/belphegor/internal/netstack"
	"github.com/labi-le/belphegor/internal/notification"
	"github.com/labi-le/belphegor/internal/store"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/network"
	"github.com/rs/zerolog"
)

type Options struct {
	PublicPort     int
	KeepAlive      time.Duration
	Deadline       network.Deadline
	Notifier       notification.Notifier
	Discovering    DiscoverOptions
	Metadata       domain.Device
	Logger         zerolog.Logger
	Secret         string
	MaxPeers       int
	MaxReceiveSize uint64
	Store          store.FileWriter
}

type DiscoverOptions struct {
	Enable   bool
	Delay    time.Duration
	MaxPeers int
}

type Option func(*Options)

//nolint:mnd //shut up
var defaultOptions = Options{
	PublicPort: netstack.RandomPort(),
	KeepAlive:  time.Minute,
	Deadline: network.Deadline{
		Read:  5 * time.Second,
		Write: 5 * time.Second,
	},
	Notifier: new(notification.BeepDecorator),
	Discovering: DiscoverOptions{
		Enable:   true,
		Delay:    5 * time.Minute,
		MaxPeers: 5,
	},
	Metadata: domain.SelfMetaData(),
}

func NewOptions(opts ...Option) Options {
	options := defaultOptions

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

func WithKeepAlive(duration time.Duration) Option {
	return func(o *Options) {
		o.KeepAlive = duration
	}
}

func WithDeadline(dd network.Deadline) Option {
	return func(o *Options) {
		o.Deadline = dd
	}
}

func WithNotifier(notifier notification.Notifier) Option {
	return func(o *Options) {
		o.Notifier = notifier
	}
}

func WithDiscovering(opt DiscoverOptions) Option {
	return func(o *Options) {
		o.Discovering = opt
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

func WithSecret(secret string) Option {
	return func(options *Options) {
		options.Secret = secret
	}
}

func WithMaxPeers(peers int) Option {
	return func(options *Options) {
		options.MaxPeers = peers
	}
}

func WithMaxReceiveSize(size uint64) Option {
	return func(options *Options) {
		options.MaxReceiveSize = size
	}
}

func WithFileStore(store store.FileWriter) Option {
	return func(options *Options) {
		options.Store = store
	}
}
