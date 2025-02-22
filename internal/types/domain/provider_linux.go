package domain

import (
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/clipboard/generic"
	"github.com/rs/zerolog/log"
)

var (
	CurrentClipboardProvider = ClipboardProviderFromManager(clipboard.New())
)

type ClipboardProvider uint32

const (
	ClipboardNull ClipboardProvider = iota
	ClipboardXSel
	ClipboardXClip
	ClipboardWlClipboard
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
	case generic.WlClipboard:
		return ClipboardWlClipboard
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
