//go:build unix

package wlr

import (
	"errors"
	"fmt"

	wl "deedles.dev/wl/client"
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/rs/zerolog"
)

type (
	controlManager = ExtDataControlManagerV1
	controlDevice  = ExtDataControlDeviceV1
	controlOffer   = ExtDataControlOfferV1
	controlSource  = ExtDataControlSourceV1
)

var (
	managerInterface = ExtDataControlManagerV1Interface
	binder           = BindExtDataControlManagerV1
)

type preset struct {
	client        *wl.Client
	registry      *wl.Registry
	seat          *wl.Seat
	deviceManager *controlManager
	display       *wl.Display
	device        *controlDevice
	logger        zerolog.Logger
	opts          eventful.Options
}

func newPreset(client *wl.Client, log zerolog.Logger, opts eventful.Options) *preset {
	return &preset{
		client: client,
		logger: log.With().Str("component", "preset").Logger(),
		opts:   opts,
	}
}

func (ws *preset) Global(name uint32, inter string, version uint32) {
	switch inter {
	case wl.SeatInterface:
		ws.seat = wl.BindSeat(ws.client, ws.registry, name, version)
		ws.logger.Trace().Type("bound seat", ws.seat).Send()
	case managerInterface:
		ws.deviceManager = binder(ws.client, ws.registry, name, version)
		ws.logger.Trace().Type("bound data device manager", ws.deviceManager).Send()
	}
}

func (ws *preset) GlobalRemove(uint32) {}

func (ws *preset) Setup() error {
	ws.display = ws.client.Display()
	ws.registry = ws.display.GetRegistry()
	ws.registry.Listener = ws

	err := ws.client.RoundTrip()
	if err != nil {
		return fmt.Errorf("round trip: %w", err)
	}
	if ws.seat == nil {
		return errors.New("no seat found")
	}
	if ws.deviceManager == nil {
		return fmt.Errorf("your wayland composer doesn't support protocol: %s", managerInterface)
	}

	ws.device = ws.deviceManager.GetDataDevice(ws.seat)
	return nil
}
