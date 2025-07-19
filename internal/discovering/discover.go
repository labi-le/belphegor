package discovering

import (
	"context"
	"fmt"
	"github.com/labi-le/belphegor/internal/node"
	"github.com/labi-le/belphegor/internal/types/domain"
	proto2 "github.com/labi-le/belphegor/internal/types/proto"
	"github.com/labi-le/belphegor/pkg/ctxlog"
	"github.com/labi-le/belphegor/pkg/ip"
	"github.com/schollz/peerdiscovery"
	"google.golang.org/protobuf/proto"
	"net"
	"time"
)

type Discover struct {
	maxPeers int
	delay    time.Duration
	port     int
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
	ctxLog := ctxlog.Op("discover.Discover")
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

				var msg proto2.GreetMessage
				if protoErr := proto.Unmarshal(d.Payload, &msg); protoErr != nil {
					ctxLog.Err(protoErr).Msg("failed to unmarshal payload")
					return
				}
				greet := domain.GreetFromProto(&msg)

				ctxLog.Trace().
					Str("peer", greet.MetaData.String()).
					Str("address", peerIP.String()).
					Uint32("port", greet.Port).
					Msg("discovered")

				go n.ConnectTo(
					ctx,
					fmt.Sprintf(
						"%s:%d",
						peerIP.String(),
						greet.Port,
					))
			},
		},
	)

	if err != nil {
		ctxLog.Fatal().Err(err).Msg("failed to start discover")
	}
}

func createPayload(metadata domain.MetaData, port int) []byte {
	greet := domain.NewGreet(domain.WithMetadata(metadata))
	greet.Port = uint32(port)
	byt, _ := proto.Marshal(greet.Proto())
	return byt
}
