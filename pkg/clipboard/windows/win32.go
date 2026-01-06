//go:build windows

package windows

import "syscall"

const (
	wmClipboardUpdate = 0x031D
	wmDestroy         = 0x0002
	className         = "BelphegorClipboardListener"
)

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   syscall.Handle
	Icon       syscall.Handle
	Cursor     syscall.Handle
	Background syscall.Handle
	MenuName   *uint16
	ClassName  *uint16
	IconSm     syscall.Handle
}

const (
	cFmtUnicodeText = 13
	cFmtDIBV5       = 17
	cFmtHDrop       = 15
)

const (
	gmemMoveable = 0x0002
)

// Win32 API
var (
	user32   = syscall.MustLoadDLL("user32")
	shell32  = syscall.MustLoadDLL("shell32")
	kernel32 = syscall.NewLazyDLL("kernel32")

	openClipboard    = user32.MustFindProc("OpenClipboard")
	closeClipboard   = user32.MustFindProc("CloseClipboard")
	emptyClipboard   = user32.MustFindProc("EmptyClipboard")
	getClipboardData = user32.MustFindProc("GetClipboardData")
	setClipboardData = user32.MustFindProc("SetClipboardData")

	addClipboardFormatListener    = user32.MustFindProc("AddClipboardFormatListener")
	removeClipboardFormatListener = user32.MustFindProc("RemoveClipboardFormatListener")
	createWindowEx                = user32.MustFindProc("CreateWindowExW")
	defWindowProc                 = user32.MustFindProc("DefWindowProcW")
	registerClassEx               = user32.MustFindProc("RegisterClassExW")
	getMessage                    = user32.MustFindProc("GetMessageW")
	dispatchMessage               = user32.MustFindProc("DispatchMessageW")
	translateMessage              = user32.MustFindProc("TranslateMessage")
	postQuitMessage               = user32.MustFindProc("PostQuitMessage")
	destroyWindow                 = user32.MustFindProc("DestroyWindow")
	postMessage                   = user32.MustFindProc("PostMessageW")

	dragQueryFileW = shell32.MustFindProc("DragQueryFileW")

	gLock   = kernel32.NewProc("GlobalLock")
	gUnlock = kernel32.NewProc("GlobalUnlock")
	gAlloc  = kernel32.NewProc("GlobalAlloc")
	gFree   = kernel32.NewProc("GlobalFree")

	getPriorityClipboardFormat = user32.MustFindProc("GetPriorityClipboardFormat")
)
