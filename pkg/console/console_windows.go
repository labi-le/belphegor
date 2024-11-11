//go:build windows

package console

import "golang.org/x/sys/windows"

var (
	kernel32         = windows.NewLazyDLL("kernel32.dll")
	getConsoleWindow = kernel32.NewProc("GetConsoleWindow")
	user32           = windows.NewLazyDLL("user32.dll")
	showWindow       = user32.NewProc("ShowWindow")
)

func HideConsoleWindow() {
	hwnd, _, _ := getConsoleWindow.Call()
	if hwnd != 0 {
		_, _, _ = showWindow.Call(hwnd, 0)
	}
}
