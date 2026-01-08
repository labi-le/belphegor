package discovering

import (
	"context"
	"net"
	"time"

	"github.com/labi-le/belphegor/pkg/ctxlog"
	"github.com/labi-le/belphegor/pkg/network"
	"github.com/rs/zerolog"
	"github.com/schollz/peerdiscovery"
)

type Connector interface {
	DiscoveryPayload() []byte
	PeerDiscovered(ctx context.Context, addr net.IP, payload []byte)
}

type Discover struct {
	maxPeers int
	delay    time.Duration
	port     int
	logger   zerolog.Logger
}

//nolint:mnd //shut up
var defaultConfig = &Discover{
	maxPeers: 10,
	delay:    time.Minute * 5,
}

type Option func(*Discover)

func WithMaxPeers(maxPeers int) Option {
	return func(d *Discover) {
		d.maxPeers = maxPeers
	}
}

func WithDelay(delay time.Duration) Option {
	return func(d *Discover) {
		d.delay = delay
	}
}

func WithPort(port int) Option {
	return func(d *Discover) {
		d.port = port
	}
}

func WithLogger(logger zerolog.Logger) Option {
	return func(d *Discover) {
		d.logger = logger
	}
}

func New(opts ...Option) *Discover {
	d := &Discover{
		maxPeers: defaultConfig.maxPeers,
		delay:    defaultConfig.delay,
		port:     defaultConfig.port,
	}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

func (d *Discover) Discover(ctx context.Context, connector Connector) {
	ctxLog := ctxlog.Op(d.logger, "discover.Discover")
	_, err := peerdiscovery.NewPeerDiscovery(
		peerdiscovery.Settings{
			Payload:   connector.DiscoveryPayload(),
			Limit:     d.maxPeers,
			TimeLimit: -1,
			Delay:     d.delay,
			AllowSelf: false,
			Notify: func(d peerdiscovery.Discovered) {
				peerIP := net.ParseIP(d.Address)
				// For some reason the library calls Notify ignoring AllowSelf:false
				if network.IsLocalIP(peerIP) {
					return
				}

				go connector.PeerDiscovered(ctx, peerIP, d.Payload)
			},
		},
	)

	if err != nil {
		ctxLog.Fatal().Err(err).Msg("failed to start discover")
	}
}
