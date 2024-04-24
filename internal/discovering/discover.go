package discovering

import (
	"fmt"
	"github.com/labi-le/belphegor/internal/node"
	"github.com/labi-le/belphegor/internal/node/data"
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

func New(maxPeers int, delay time.Duration, port int) *Discover {
	return &Discover{
		maxPeers: maxPeers,
		delay:    delay,
		port:     port,
	}
}

func (d *Discover) Discover(n *node.Node) {
	_, err := peerdiscovery.NewPeerDiscovery(
		peerdiscovery.Settings{
			PayloadFunc: func() []byte {
				greet := data.NewGreet(n.Metadata())
				defer greet.Release()

				greet.Port = uint32(d.port)

				byt, _ := proto.Marshal(greet)
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

				greet := data.NewGreet(n.Metadata())
				defer greet.Release()

				if protoErr := proto.Unmarshal(d.Payload, greet); protoErr != nil {
					log.Error().Err(protoErr).Msg("failed to unmarshal payload")
					return
				}

				peerAddr := fmt.Sprintf(
					"%s:%s",
					peerIP.String(),
					strconv.Itoa(int(greet.Port)),
				)

				go n.ConnectTo(peerAddr)
			},
		},
	)

	if err != nil {
		log.Fatal().Err(err).Msg("failed to discover nodes")
	}
}
