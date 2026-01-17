package x11

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xfixes"
	"github.com/jezek/xgb/xproto"
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/mime"
	"github.com/rs/zerolog"
)

const (
	maxPropSize = 0x10000
	maxDataSize = 50 * 1024 * 1024

	xFixesClientMajor = 5
	xFixesClientMinor = 0
)

var _ eventful.Eventful = (*Clipboard)(nil)

type Clipboard struct {
	logger zerolog.Logger
	conn   *xgb.Conn
	win    xproto.Window
	atoms  *atomCache
	opts   eventful.Options

	mu       sync.Mutex
	dedup    eventful.Deduplicator
	serving  []byte
	serveTyp xproto.Atom
}

func New(log zerolog.Logger, opts eventful.Options) *Clipboard {
	return &Clipboard{
		logger: log.With().Str("component", "x11").Logger(),
		opts:   opts,
	}
}

func (c *Clipboard) init() error {
	var err error
	if c.conn, err = xgb.NewConn(); err != nil {
		return fmt.Errorf("xgb connect: %w", err)
	}

	if err := xfixes.Init(c.conn); err != nil {
		return fmt.Errorf("xfixes init: %w", err)
	}

	if _, err := xfixes.QueryVersion(c.conn, xFixesClientMajor, xFixesClientMinor).Reply(); err != nil {
		return fmt.Errorf("xfixes query version: %w", err)
	}

	if c.atoms, err = loadAtoms(c.conn); err != nil {
		return fmt.Errorf("load atoms: %w", err)
	}

	screen := xproto.Setup(c.conn).DefaultScreen(c.conn)
	if c.win, err = xproto.NewWindowId(c.conn); err != nil {
		return err
	}

	err = xproto.CreateWindowChecked(
		c.conn,
		screen.RootDepth,
		c.win,
		screen.Root,
		0,
		0,
		1,
		1,
		0,
		xproto.WindowClassInputOutput,
		screen.RootVisual,
		xproto.CwEventMask,
		[]uint32{xproto.EventMaskPropertyChange},
	).Check()
	if err != nil {
		return fmt.Errorf("create window: %w", err)
	}

	mask := xfixes.SelectionEventMaskSetSelectionOwner |
		xfixes.SelectionEventMaskSelectionWindowDestroy |
		xfixes.SelectionEventMaskSelectionClientClose
	err = xfixes.SelectSelectionInputChecked(c.conn, c.win, c.atoms.Clipboard, uint32(mask)).Check()
	if err != nil {
		return fmt.Errorf("select selection input: %w", err)
	}

	return nil
}

func (c *Clipboard) Watch(ctx context.Context, upd chan<- eventful.Update) error {
	defer close(upd)

	if c.conn == nil {
		if err := c.init(); err != nil {
			return err
		}
	}
	defer c.conn.Close()

	go c.fetch()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			ev, err := c.conn.WaitForEvent()
			if err != nil {
				return err
			}
			if ev == nil {
				continue
			}
			c.handleEvent(ev, upd)
		}
	}
}

func (c *Clipboard) handleEvent(ev xgb.Event, upd chan<- eventful.Update) {
	switch e := ev.(type) {
	case xfixes.SelectionNotifyEvent:
		if e.Owner != c.win && e.Selection == c.atoms.Clipboard {
			c.fetch()
		}
	case xproto.SelectionRequestEvent:
		c.handleRequest(e)
	case xproto.SelectionNotifyEvent:
		c.handleNotify(e, upd)
	}
}

func (c *Clipboard) Write(t mime.Type, src []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return 0, errors.New("x11 not initialized")
	}

	c.serving = make([]byte, len(src))
	copy(c.serving, src)
	c.dedup.Mark(src)

	switch t {
	case mime.TypeImage:
		c.serveTyp = c.atoms.ImagePng
	case mime.TypePath:
		c.serveTyp = c.atoms.UriList
	default:
		c.serveTyp = c.atoms.Utf8String
	}

	err := xproto.SetSelectionOwnerChecked(c.conn, c.win, c.atoms.Clipboard, xproto.TimeCurrentTime).Check()
	if err != nil {
		return 0, fmt.Errorf("set selection owner: %w", err)
	}

	return len(src), nil
}

func (c *Clipboard) handleRequest(e xproto.SelectionRequestEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()

	resp := xproto.SelectionNotifyEvent{
		Time:      e.Time,
		Requestor: e.Requestor,
		Selection: e.Selection,
		Target:    e.Target,
		Property:  xproto.AtomNone,
	}

	reply := func(prop xproto.Atom, typ xproto.Atom, fmt uint8, data []byte) {
		xproto.ChangeProperty(c.conn, xproto.PropModeReplace, e.Requestor, prop, typ, fmt, uint32(len(data))/(uint32(fmt)/8), data)
		resp.Property = prop
	}

	switch e.Target {
	case c.atoms.Targets:
		targets := []xproto.Atom{c.atoms.Targets, c.atoms.Timestamp, c.atoms.SaveTargets, c.serveTyp}
		if c.serveTyp == c.atoms.Utf8String || c.serveTyp == c.atoms.UriList {
			targets = append(targets, c.atoms.Utf8String, c.atoms.String)
		}

		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, targets)
		reply(e.Property, xproto.AtomAtom, 32, buf.Bytes())

	case c.atoms.Timestamp:
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, e.Time)
		reply(e.Property, xproto.AtomInteger, 32, buf.Bytes())

	case c.atoms.SaveTargets, c.atoms.Delete:
		resp.Property = e.Property

	default:
		if e.Target == c.serveTyp {
			reply(e.Property, e.Target, 8, c.serving)
			break
		}
		isTextReq := e.Target == c.atoms.Utf8String || e.Target == c.atoms.String
		isTextSrv := c.serveTyp == c.atoms.Utf8String || c.serveTyp == c.atoms.UriList
		if isTextReq && isTextSrv {
			reply(e.Property, e.Target, 8, c.serving)
		}
	}

	buf := new(bytes.Buffer)
	buf.WriteByte(31)
	buf.WriteByte(0)
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))
	_ = binary.Write(buf, binary.LittleEndian, resp.Time)
	_ = binary.Write(buf, binary.LittleEndian, resp.Requestor)
	_ = binary.Write(buf, binary.LittleEndian, resp.Selection)
	_ = binary.Write(buf, binary.LittleEndian, resp.Target)
	_ = binary.Write(buf, binary.LittleEndian, resp.Property)
	buf.Write(make([]byte, 8))

	xproto.SendEvent(c.conn, false, e.Requestor, xproto.EventMaskNoEvent, string(buf.Bytes()))
}

func (c *Clipboard) fetch() {
	xproto.ConvertSelection(c.conn, c.win, c.atoms.Clipboard, c.atoms.Targets, c.atoms.LocalProp, xproto.TimeCurrentTime)
}

func (c *Clipboard) handleNotify(e xproto.SelectionNotifyEvent, upd chan<- eventful.Update) {
	if e.Property == xproto.AtomNone {
		return
	}

	if e.Target == c.atoms.Targets {
		prop, err := xproto.GetProperty(c.conn, false, c.win, e.Property, xproto.AtomAtom, 0, 1024).Reply()
		if err != nil || prop.Format != 32 {
			return
		}

		ids := make([]xproto.Atom, prop.ValueLen)
		_ = binary.Read(bytes.NewReader(prop.Value), binary.LittleEndian, &ids)

		var requestFormat xproto.Atom

		hasAtom := func(target xproto.Atom) bool {
			for _, id := range ids {
				if id == target {
					return true
				}
			}
			return false
		}

		if hasAtom(c.atoms.ImagePng) {
			requestFormat = c.atoms.ImagePng
		} else if hasAtom(c.atoms.UriList) {
			requestFormat = c.atoms.UriList
		} else if hasAtom(c.atoms.Utf8String) {
			requestFormat = c.atoms.Utf8String
		} else if hasAtom(c.atoms.String) {
			requestFormat = c.atoms.String
		}

		if requestFormat != 0 {
			xproto.ConvertSelection(c.conn, c.win, c.atoms.Clipboard, requestFormat, c.atoms.LocalProp, xproto.TimeCurrentTime)
		}
		return
	}

	reply, err := xproto.GetProperty(c.conn, false, c.win, e.Property, xproto.GetPropertyTypeAny, 0, 0).Reply()
	if err != nil {
		return
	}

	var data []byte

	if reply.Type == c.atoms.Incr {
		data, err = c.readIncr(e.Property)
		if err != nil {
			c.logger.Error().Err(err).Msg("failed to read INCR data")
			return
		}
	} else {
		fullReply, err := xproto.GetProperty(
			c.conn,
			true,
			c.win,
			e.Property,
			xproto.GetPropertyTypeAny,
			0,
			maxPropSize,
		).Reply()
		if err != nil {
			return
		}
		data = fullReply.Value
	}

	if len(data) == 0 {
		return
	}

	if e.Target == c.atoms.UriList {
		if !c.opts.AllowCopyFiles {
			return
		}

		updates, batchHash := eventful.UpdatesFromRawPath(data, c.opts.MaxClipboardFiles)
		if len(updates) == 0 {
			return
		}

		if _, ok := c.dedup.Check(batchHash); ok {
			for _, u := range updates {
				upd <- u
			}
		}
		return
	}

	if h, ok := c.dedup.Check(data); ok {
		var mTyp mime.Type
		switch e.Target {
		case c.atoms.ImagePng:
			mTyp = mime.TypeImage
		default:
			mTyp = mime.TypeText
		}

		upd <- eventful.Update{
			Data:     data,
			MimeType: mTyp,
			Hash:     h,
		}
	}
}

func (c *Clipboard) readIncr(prop xproto.Atom) ([]byte, error) {
	xproto.DeleteProperty(c.conn, c.win, prop)

	var buf bytes.Buffer
	buf.Grow(4096)

	for {
		ev, err := c.conn.WaitForEvent()
		if err != nil {
			return nil, err
		}

		switch event := ev.(type) {
		case xproto.PropertyNotifyEvent:
			if event.Window != c.win || event.Atom != prop || event.State != xproto.PropertyNewValue {
				continue
			}

			reply, err := xproto.GetProperty(c.conn, true, c.win, prop, xproto.GetPropertyTypeAny, 0, maxPropSize).Reply()
			if err != nil {
				return nil, err
			}

			if len(reply.Value) == 0 {
				return buf.Bytes(), nil
			}

			if buf.Len()+len(reply.Value) > maxDataSize {
				return nil, errors.New("clipboard data exceeded limit")
			}

			buf.Write(reply.Value)
		}
	}
}
