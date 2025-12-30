package node

import (
	"time"

	"github.com/labi-le/belphegor/internal/netstack"
	"github.com/labi-le/belphegor/internal/notification"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/rs/zerolog"
)

type Options struct {
	PublicPort         int
	BitSize            int
	KeepAlive          time.Duration
	ClipboardScanDelay time.Duration
	WriteTimeout       time.Duration
	Notifier           notification.Notifier
	Discovering        DiscoverOptions
	Metadata           domain.Device
	Logger             zerolog.Logger
}

type DiscoverOptions struct {
	Enable   bool
	Delay    time.Duration
	MaxPeers int
}

// Option defines the method to configure Options
type Option func(*Options)

var defaultOptions = Options{
	PublicPort:         netstack.RandomPort(),
	BitSize:            2048,
	KeepAlive:          time.Minute,
	ClipboardScanDelay: 2 * time.Second,
	WriteTimeout:       5 * time.Second,
	Notifier:           new(notification.BeepDecorator),
	Discovering: DiscoverOptions{
		Enable:   true,
		Delay:    5 * time.Minute,
		MaxPeers: 5,
	},
	Metadata: domain.SelfMetaData(),
}

// NewOptions creates Options with provided options
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

func WithBitSize(size int) Option {
	return func(o *Options) {
		o.BitSize = size
	}
}

func WithKeepAlive(duration time.Duration) Option {
	return func(o *Options) {
		o.KeepAlive = duration
	}
}

func WithClipboardScanDelay(delay time.Duration) Option {
	return func(o *Options) {
		o.ClipboardScanDelay = delay
	}
}

func WithWriteTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.WriteTimeout = timeout
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
