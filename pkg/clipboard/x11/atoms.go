package x11

import (
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

type atomCache struct {
	Clipboard   xproto.Atom
	Targets     xproto.Atom
	Timestamp   xproto.Atom
	SaveTargets xproto.Atom
	Delete      xproto.Atom
	Incr        xproto.Atom
	Utf8String  xproto.Atom
	String      xproto.Atom
	ImagePng    xproto.Atom
	UriList     xproto.Atom
	LocalProp   xproto.Atom
}

func loadAtoms(c *xgb.Conn) (*atomCache, error) {
	names := []string{
		"CLIPBOARD", "TARGETS", "TIMESTAMP", "SAVE_TARGETS", "DELETE", "INCR",
		"UTF8_STRING", "STRING", "image/png", "text/uri-list",
		"BELPHEGOR_SELECTION",
	}

	cookies := make([]xproto.InternAtomCookie, len(names))
	for i, name := range names {
		cookies[i] = xproto.InternAtom(c, false, uint16(len(name)), name)
	}

	atoms := make([]xproto.Atom, len(names))
	for i, cookie := range cookies {
		reply, err := cookie.Reply()
		if err != nil {
			return nil, err
		}
		atoms[i] = reply.Atom
	}

	return &atomCache{
		Clipboard:   atoms[0],
		Targets:     atoms[1],
		Timestamp:   atoms[2],
		SaveTargets: atoms[3],
		Delete:      atoms[4],
		Incr:        atoms[5],
		Utf8String:  atoms[6],
		String:      atoms[7],
		ImagePng:    atoms[8],
		UriList:     atoms[9],
		LocalProp:   atoms[10],
	}, nil
}
