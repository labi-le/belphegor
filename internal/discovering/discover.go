package discovering

import (
	"fmt"
	"github.com/labi-le/belphegor/internal/node"
	"github.com/labi-le/belphegor/internal/types/domain"
	proto2 "github.com/labi-le/belphegor/internal/types/proto"
	"github.com/labi-le/belphegor/pkg/ip"
	"github.com/rs/zerolog/log"
	"github.com/schollz/peerdiscovery"
	"google.golang.org/protobuf/proto"
	"net"
	"strconv"
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

func (d *Discover) Discover(n *node.Node) {
	ctxLog := log.With().Str("op", "discover.Discover").Logger()
	_, err := peerdiscovery.NewPeerDiscovery(
		peerdiscovery.Settings{
			PayloadFunc: func() []byte {
				greet := domain.NewGreet()
				greet.Port = uint32(d.port)
				byt, _ := proto.Marshal(greet.Proto())
				return byt
			},
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

				ctxLog.Info().
					Str("op", "discovering").
					Str("peer", greet.MetaData.String()).
					Str("address", peerIP.String()).
					Uint32("port", greet.Port).
					Msg("discovered new peer")

				go n.ConnectTo(fmt.Sprintf(
					"%s:%s",
					peerIP.String(),
					strconv.Itoa(int(greet.Port)),
				))
			},
		},
	)

	if err != nil {
		log.Fatal().Err(err).Msg("failed to discover nodes")
	}
}
