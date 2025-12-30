//go:build unix

package domain

import (
	"fmt"

	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/clipboard/mac"
	"github.com/labi-le/belphegor/pkg/clipboard/null"
	"github.com/labi-le/belphegor/pkg/clipboard/wlclipboard"
	"github.com/labi-le/belphegor/pkg/clipboard/wlr"
	"github.com/labi-le/belphegor/pkg/clipboard/xclip"
	"github.com/labi-le/belphegor/pkg/clipboard/xsel"
	"github.com/rs/zerolog/log"
)

var (
	CurrentClipboardProvider = ClipboardProviderFromManager(clipboard.New(log.Logger))
)

type ClipboardProvider int32

const (
	ClipboardNull ClipboardProvider = iota
	ClipboardXSel
	ClipboardXClip
	ClipboardWlClipboard
	ClipboardMasOsStd
)

func ClipboardProviderFromManager(m eventful.Eventful) ClipboardProvider {
	switch m.(type) {
	case *xsel.Clipboard:
		return ClipboardXSel
	case *xclip.Clipboard:
		return ClipboardXClip
	case *wlclipboard.Clipboard:
		return ClipboardWlClipboard
	case *wlr.Clipboard:
		return ClipboardWlClipboard
	case *mac.Clipboard:
		return ClipboardMasOsStd
	case *null.Clipboard:
		return ClipboardNull
	default:
		panic(fmt.Errorf("unimplemented provider: %T", m))
	}

	// unreachable
	return ClipboardNull
}
