package node

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/labi-le/belphegor/internal/netstack"
	"github.com/labi-le/belphegor/internal/notification"
	"github.com/labi-le/belphegor/internal/store"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/network"
	"github.com/rs/zerolog"
)

type Transport string

const (
	TransportQUIC Transport = "quic"
	TransportTCP  Transport = "tcp"
)

func (t Transport) String() string {
	return string(t)
}

func (t *Transport) Set(s string) error {
	*t = Transport(s)
	if !t.valid() {
		return fmt.Errorf(
			"available %s, %s",
			TransportTCP,
			TransportQUIC,
		)
	}
	return nil
}

func (t *Transport) Type() string {
	return "string"
}

func (t *Transport) valid() bool {
	return *t == TransportTCP || *t == TransportQUIC
}

type Options struct {
	ListenPort  int
	Transport   Transport
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

	FileSavePath   string
	Verbose        bool
	Notify         bool
	ShowVersion    bool
	ShowHelp       bool
	Hidden         bool
	InstallService bool
}

func (o Options) MarshalZerologObject(e *zerolog.Event) {
	e.Int("public_port", int(o.ListenPort))
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

func DefaultOptions() Options {
	return Options{
		ListenPort: netstack.RandomPort(),
		Transport:  TransportQUIC,
		KeepAlive:  time.Minute,
		Deadline: network.Deadline{
			Read:  time.Minute,
			Write: time.Minute,
		},
		Notifier: new(notification.BeepDecorator),
		Discovering: DiscoverOptions{
			Enable:   true,
			Delay:    30 * time.Second,
			MaxPeers: 10,
		},
		Metadata:     domain.SelfMetaData(),
		MaxPeers:     10,
		FileSavePath: path.Join(os.TempDir(), "bfg_cache"),
		Clip: eventful.Options{
			AllowCopyFiles: true,
			// 512 mb
			MaxFileSize:       1 << 29,
			MaxClipboardFiles: 15,
		},
	}
}

func (o Options) Validated() Options {
	var defaults = DefaultOptions()
	if o.ListenPort <= 0 || o.ListenPort > 65535 {
		o.ListenPort = defaults.ListenPort
	}

	if !o.Transport.valid() {
		o.Transport = defaults.Transport
	}

	if o.KeepAlive <= 0 {
		o.KeepAlive = defaults.KeepAlive
	}

	if o.Deadline.Write <= 0 {
		o.Deadline.Write = defaults.Deadline.Write
	}

	if o.Deadline.Read <= 0 {
		o.Deadline.Read = defaults.Deadline.Read
	}

	if o.Notifier == nil {
		o.Notifier = defaults.Notifier
	}

	if o.Discovering.MaxPeers <= 0 {
		o.Discovering.MaxPeers = defaults.Discovering.MaxPeers
	}

	if o.Discovering.Delay <= 0 {
		o.Discovering.Delay = defaults.Discovering.Delay
	}

	if o.Metadata == (domain.Device{}) {
		o.Metadata = defaults.Metadata
	}

	if o.MaxPeers <= 0 {
		o.MaxPeers = defaults.MaxPeers
	}

	if o.Clip.MaxClipboardFiles <= 0 {
		o.Clip.MaxClipboardFiles = defaults.Clip.MaxClipboardFiles
	}

	if o.Clip.MaxClipboardFiles <= 0 {
		o.Clip.MaxClipboardFiles = defaults.Clip.MaxClipboardFiles
	}

	return o
}
