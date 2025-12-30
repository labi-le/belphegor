//go:build windows

package domain

import (
	"fmt"

	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/clipboard/null"
	"github.com/labi-le/belphegor/pkg/clipboard/windows"
	"github.com/rs/zerolog"
)

var (
	CurrentClipboardProvider = ClipboardProviderFromManager(clipboard.New(zerolog.Nop()))
)

type ClipboardProvider int32

const (
	ClipboardNull ClipboardProvider = iota
	_
	_
	_
	_
	ClipboardWindowsNT10
)

func ClipboardProviderFromManager(m eventful.Eventful) ClipboardProvider {
	switch m.(type) {
	case *windows.Clipboard:
		return ClipboardWindowsNT10
	case *null.Clipboard:
		return ClipboardNull
	default:
		panic(fmt.Errorf("unimplemented provider: %T", m))
	}

	// unreachable
	return ClipboardNull
}
