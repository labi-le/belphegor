//go:build unix

package console

import (
	"context"
)

func HideConsoleWindow(context.CancelFunc) {
	// unix-like users don't need it
}
