package discovering

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/labi-le/belphegor/internal/node"
	"github.com/labi-le/belphegor/internal/types/domain"
	proto2 "github.com/labi-le/belphegor/internal/types/proto"
	"github.com/labi-le/belphegor/pkg/ctxlog"
	"github.com/labi-le/belphegor/pkg/ip"
	"github.com/rs/zerolog"
	"github.com/schollz/peerdiscovery"
	"google.golang.org/protobuf/proto"
)

type Discover struct {
	maxPeers int
	delay    time.Duration
	port     int
	logger   zerolog.Logger
}

var defaultConfig = &Discover{
	maxPeers: 10,
	delay:    time.Minute * 5,
}

// Option defines the method to configure Discover
type Option func(*Discover)

// WithMaxPeers sets the maximum number of peers
func WithMaxPeers(maxPeers int) Option {
	return func(d *Discover) {
		d.maxPeers = maxPeers
	}
}

// WithDelay sets the delay between discovery attempts
func WithDelay(delay time.Duration) Option {
	return func(d *Discover) {
		d.delay = delay
	}
}

// WithPort sets the port for peer discovery
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

// New creates a new Discover instance with the provided options
func New(opts ...Option) *Discover {
	d := &Discover{
		maxPeers: defaultConfig.maxPeers,
		delay:    defaultConfig.delay,
		port:     defaultConfig.port,
	}

	// Apply the options
	for _, opt := range opts {
		opt(d)
	}

	return d
}

func (d *Discover) Discover(ctx context.Context, n *node.Node) {
	ctxLog := ctxlog.Op(d.logger, "discover.Discover")
	_, err := peerdiscovery.NewPeerDiscovery(
		peerdiscovery.Settings{
			Payload:   createPayload(n.Metadata(), d.port),
			Limit:     d.maxPeers,
			TimeLimit: -1,
			Delay:     d.delay,
			AllowSelf: false,
			Notify: func(d peerdiscovery.Discovered) {
				peerIP := net.ParseIP(d.Address)
				// For some reason the library calls Notify ignoring AllowSelf:false
				if ip.IsLocalIP(peerIP) {
					return
				}

				var msg proto2.Event
				if protoErr := proto.Unmarshal(d.Payload, &msg); protoErr != nil {
					ctxLog.Err(protoErr).Msg("failed to unmarshal payload")
					return
				}
				greet := domain.GreetFromProto(&msg)

				ctxLog.Trace().
					Str("peer", greet.Payload.MetaData.String()).
					Str("address", peerIP.String()).
					Uint32("port", greet.Payload.Port).
					Msg("discovered")

				go n.ConnectTo(ctx, createConnDsn(peerIP, greet))
			},
		},
	)

	if err != nil {
		ctxLog.Fatal().Err(err).Msg("failed to start discover")
	}
}

func createConnDsn(peerIP net.IP, greet domain.EventHandshake) string {
	return fmt.Sprintf(
		"%s:%d",
		peerIP.String(),
		greet.Payload.Port,
	)
}

func createPayload(metadata domain.Device, port int) []byte {
	greet := domain.NewGreet(domain.WithMetadata(metadata))
	greet.Payload.Port = uint32(port)
	byt, _ := proto.Marshal(greet.Proto())
	return byt
}
