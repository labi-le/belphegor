//go:build linux

package clipboard

import (
	"errors"
	"github.com/labi-le/belphegor/pkg/clipboard/api"
	"github.com/labi-le/belphegor/pkg/clipboard/generic"
	"github.com/labi-le/belphegor/pkg/clipboard/wlr"
	"github.com/rs/zerolog"
	"os/exec"
)

var ErrUnimplementedClipboardManager = errors.New("unimplemented clipboard wrapper")

var managers = []struct {
	manager func(log zerolog.Logger) api.Eventful
	check   func() bool
}{
	{func(log zerolog.Logger) api.Eventful { return generic.XClip{} }, func() bool { return binExist("xclip") }},
	{func(log zerolog.Logger) api.Eventful { return generic.XSel{} }, func() bool { return binExist("xsel") }},
	{func(log zerolog.Logger) api.Eventful { return wlr.Must(log) }, func() bool { return wlr.Supported }},
	{func(log zerolog.Logger) api.Eventful { return generic.Termux{} }, func() bool { return binExist("termux-api") }},
}

func binExist(name string) bool {
	if _, err := exec.LookPath(name); err != nil {
		return false
	}

	return true
}

func findClipboardManager(log zerolog.Logger) api.Eventful {
	for _, manager := range managers {
		if manager.check() {
			return manager.manager(log)
		}
	}

	panic(ErrUnimplementedClipboardManager)
}

func New(log zerolog.Logger) api.Eventful {
	return findClipboardManager(log)
}
