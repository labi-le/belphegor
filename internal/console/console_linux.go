//go:build linux

package console

import (
	"context"
	_ "image/png"
	"sync/atomic"

	"fyne.io/systray"
	"github.com/labi-le/belphegor/internal/console/icon"
)

type appState struct {
	isRedIcon atomic.Bool
	cancel    context.CancelFunc
}

func HideConsoleWindow(cancel context.CancelFunc) {
	app := &appState{
		cancel: cancel,
	}

	systray.Run(app.onReady, app.onExit)
}

func (app *appState) onReady() {
	systray.SetIcon(icon.RedPNG)
	systray.SetTitle("belphegor")
	systray.SetOnTapped(app.toggleIconColor)

	mQuit := systray.AddMenuItem("Quit", "")
	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
	}()
}

func (app *appState) onExit() {
	if app.cancel != nil {
		app.cancel()
	}
}

func (app *appState) toggleIconColor() {
	if !app.isRedIcon.Load() {
		systray.SetIcon(icon.GreenPNG)
		app.isRedIcon.Store(true)
		return
	}

	systray.SetIcon(icon.RedPNG)
	app.isRedIcon.Store(false)
}
