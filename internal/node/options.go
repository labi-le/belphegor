package node

import (
	"github.com/labi-le/belphegor/internal/netstack"
	"github.com/labi-le/belphegor/internal/node/data"
	"time"
)

type Options struct {
	PublicPort         int
	KeepAlive          time.Duration
	ClipboardScanDelay time.Duration
	WriteTimeout       time.Duration
	// represents the current device
	Metadata *data.MetaData

	Discovering DiscoverOptions
	Encryption  EncryptionOptions
}

type DiscoverOptions struct {
	Enable      bool
	MaxPeers    int
	SearchDelay time.Duration
	Port        int
}

type EncryptionOptions struct {
	Enable bool
}

func defaultOptions() *Options {
	port := netstack.RandomPort()
	return &Options{
		PublicPort:         port,
		KeepAlive:          time.Minute,
		ClipboardScanDelay: 2 * time.Second,
		WriteTimeout:       5 * time.Second,
		Metadata:           data.SelfMetaData(),
		Discovering: DiscoverOptions{
			Enable:      true,
			MaxPeers:    5,
			SearchDelay: 5 * time.Minute,
			Port:        port,
		},
		Encryption: EncryptionOptions{
			Enable: true,
		},
	}

}
