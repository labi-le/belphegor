//go:build darwin

package domain

import (
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/clipboard/generic"
	"github.com/labi-le/belphegor/pkg/clipboard/mac"
	"github.com/labi-le/belphegor/pkg/clipboard/windows"
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
	case mac.Darwin:
		return ClipboardMasOsStd
	case windows.Windows:
		return ClipboardWindowsNT10
	case generic.Termux:
		return ClipboardTermux
	case generic.Null:
		return ClipboardNull
	default:
		log.Fatal().Msgf("unimplemented device: %+v", m)
	}

	// unreachable
	return ClipboardNull
}
