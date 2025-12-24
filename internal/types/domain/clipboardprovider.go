package domain

import (
	"fmt"

	"github.com/labi-le/belphegor/pkg/clipboard"
)

var (
	CurrentClipboardProvider = ClipboardProviderFromManager(clipboard.New())
)

type ClipboardProvider int32

const (
	ClipboardNull ClipboardProvider = iota
	ClipboardXSel
	ClipboardXClip
	ClipboardWlClipboard
	ClipboardMasOsStd
	ClipboardWindowsNT10
)

func ClipboardProviderFromManager(m clipboard.Manager) ClipboardProvider {
	switch m.Name() {
	case clipboard.XSel:
		return ClipboardXSel
	case clipboard.XClip:
		return ClipboardXClip
	case clipboard.WlClipboard:
		return ClipboardWlClipboard
	case clipboard.MasOsStd:
		return ClipboardMasOsStd
	case clipboard.WindowsNT10:
		return ClipboardWindowsNT10
	case clipboard.NullClipboard:
		return ClipboardNull
	default:
		panic(fmt.Errorf("unimplemented device: %s", m.Name()))
	}

	// unreachable
	return ClipboardNull
}
