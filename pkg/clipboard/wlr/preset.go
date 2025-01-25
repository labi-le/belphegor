package wlr

import (
	wl "deedles.dev/wl/client"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
)

type preset struct {
	client        *wl.Client
	registry      *wl.Registry
	seat          *wl.Seat
	deviceManager *ZwlrDataControlManagerV1
	display       *wl.Display
}

func (ws *preset) Global(name uint32, inter string, version uint32) {
	switch inter {
	case wl.SeatInterface:
		ws.seat = wl.BindSeat(ws.client, ws.registry, name, version)
	case ZwlrDataControlManagerV1Interface:
		ws.deviceManager = BindZwlrDataControlManagerV1(ws.client, ws.registry, name, version)
		log.Trace().Msgf("bound data device manager: %v", ws.deviceManager)
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
	return nil
}
