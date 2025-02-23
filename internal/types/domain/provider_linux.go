package domain

import (
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/clipboard/generic"
	"github.com/labi-le/belphegor/pkg/clipboard/wlr"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	CurrentClipboardProvider = ClipboardProviderFromManager(clipboard.New(zerolog.Nop()))
)

type ClipboardProvider uint32

const (
	ClipboardNull ClipboardProvider = iota
	ClipboardXSel
	ClipboardXClip
	ClipboardWlr
	ClipboardMasOsStd
	ClipboardWindowsNT10
	ClipboardTermux
)

func ClipboardProviderFromManager(m any) ClipboardProvider {
	switch m.(type) {
	case generic.XSel:
		return ClipboardXSel
	case generic.XClip:
		return ClipboardXClip
	case *wlr.Wlr:
		return ClipboardWlr
	case generic.Termux:
		return ClipboardTermux
	case *generic.Null:
		return ClipboardNull
	default:
		log.Fatal().Type("unimplemented device", m).Send()
	}

	// unreachable
	return ClipboardNull
}
