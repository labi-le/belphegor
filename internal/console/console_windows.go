//go:build windows

package console

import (
	"context"
	_ "embed"
	"sync/atomic"
	"time"

	"fyne.io/systray"
	"github.com/labi-le/belphegor/internal/console/icon"
	"golang.org/x/sys/windows"
)

const (
	SwHide  = 0
	SwShow  = 5
	GwOwner = 4
)

var (
	kernel32 = windows.NewLazyDLL("kernel32.dll")
	user32   = windows.NewLazyDLL("user32.dll")

	procGetConsoleWindow = kernel32.NewProc("GetConsoleWindow")
	procGetWindow        = user32.NewProc("GetWindow")
	procShowWindow       = user32.NewProc("ShowWindow")
)

type appState struct {
	isHidden  atomic.Bool
	isRedIcon atomic.Bool
	cancel    context.CancelFunc
}

func HideConsoleWindow(cancel context.CancelFunc) {
	app := &appState{
		cancel: cancel,
	}

	app.hideInitial()

	systray.Run(app.onReady, app.onExit)
}

func (app *appState) onReady() {
	systray.SetIcon(icon.Red)
	systray.SetTitle("belphegor")
	systray.SetOnTapped(app.toggleConsoleAndIcon)

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

func (app *appState) hideInitial() {
	time.Sleep(time.Millisecond * 50)

	hwnd := getConsoleWindowHandle()
	if hwnd == 0 {
		return
	}

	showWindow(hwnd, SwHide)
	app.isHidden.Store(true)
}

func (app *appState) toggleConsoleAndIcon() {
	hwnd := getConsoleWindowHandle()
	if hwnd != 0 {
		if app.isHidden.Load() {
			showWindow(hwnd, SwShow)
			app.isHidden.Store(false)
		} else {
			showWindow(hwnd, SwHide)
			app.isHidden.Store(true)
		}
	}

	app.toggleIconColor()
}
func (app *appState) toggleIconColor() {
	if !app.isRedIcon.Load() {
		systray.SetIcon(icon.Green)
		app.isRedIcon.Store(true)
		return
	}

	systray.SetIcon(icon.Red)
	app.isRedIcon.Store(false)
}

func getConsoleWindowHandle() uintptr {
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd == 0 {
		return 0
	}

	owner, _, _ := procGetWindow.Call(hwnd, GwOwner)
	if owner != 0 {
		return owner
	}

	return hwnd
}

func showWindow(hwnd uintptr, cmdShow int) {
	_, _, _ = procShowWindow.Call(hwnd, uintptr(cmdShow))
}
