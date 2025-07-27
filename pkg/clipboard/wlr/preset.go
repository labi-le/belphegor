package wlr

import (
	wl "deedles.dev/wl/client"
	"errors"
	"fmt"
	"github.com/rs/zerolog"
)

type preset struct {
	client        *wl.Client
	registry      *wl.Registry
	seat          *wl.Seat
	deviceManager *ZwlrDataControlManagerV1
	display       *wl.Display
	device        *ZwlrDataControlDeviceV1
	logger        zerolog.Logger
}

func newPreset(client *wl.Client, log zerolog.Logger) *preset {
	return &preset{
		client: client,
		logger: log.With().Str("component", "preset").Logger(),
	}
}

func (ws *preset) Global(name uint32, inter string, version uint32) {
	switch inter {
	case wl.SeatInterface:
		ws.seat = wl.BindSeat(ws.client, ws.registry, name, version)
		ws.logger.Trace().Type("bound seat", ws.seat)
	case ZwlrDataControlManagerV1Interface:
		ws.deviceManager = BindZwlrDataControlManagerV1(ws.client, ws.registry, name, version)
		ws.logger.Trace().Type("bound data device manager", ws.deviceManager)
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

	ws.device = ws.deviceManager.GetDataDevice(ws.seat)
	return nil
}
