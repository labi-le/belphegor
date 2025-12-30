//go:build windows

package windows

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
)

var _ eventful.Eventful = &Clipboard{}

const (
	debounceMs = 60
	timerID    = 1
)

func New() *Clipboard {
	return new(Clipboard)
}

type Clipboard struct{}

func (w *Clipboard) Watch(ctx context.Context, update chan<- eventful.Update) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hInstance, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	clsNamePtr, _ := syscall.UTF16PtrFromString(className)

	wndProc := syscall.NewCallback(func(hwnd syscall.Handle, msg uint32, wparam, lparam uintptr) uintptr {
		switch msg {
		case wmClipboardUpdate:
			killTimer.Call(uintptr(hwnd), timerID)
			setTimer.Call(uintptr(hwnd), timerID, debounceMs, 0)
			return 0

		case wmTimer:
			if wparam == timerID {
				killTimer.Call(uintptr(hwnd), timerID)

				data, det := ReadDetected(FmtFile)
				if data == nil {
					data, det = ReadDetected(FmtImage)
				}
				if data == nil {
					data, det = ReadDetected(FmtText)
				}

				if len(data) > 0 {
					select {
					case update <- eventful.Update{Data: data, MimeType: det}:
					default:
					}
				}
				return 0
			}
			return 0

		case wmDestroy:
			killTimer.Call(uintptr(hwnd), timerID)
			postQuitMessage.Call(0)
			return 0
		}

		ret, _, _ := defWindowProc.Call(uintptr(hwnd), uintptr(msg), wparam, lparam)
		return ret
	})

	wc := wndClassEx{
		Size:      uint32(unsafe.Sizeof(wndClassEx{})),
		Instance:  syscall.Handle(hInstance),
		WndProc:   wndProc,
		ClassName: clsNamePtr,
	}

	registerClassEx.Call(uintptr(unsafe.Pointer(&wc)))

	hwnd, _, _ := createWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(clsNamePtr)),
		uintptr(unsafe.Pointer(clsNamePtr)),
		0, 0, 0, 0, 0,
		0,
		0, 0, 0,
	)

	if hwnd == 0 {
		return fmt.Errorf("failed to create window listener")
	}

	ret, _, _ := addClipboardFormatListener.Call(hwnd)
	if ret == 0 {
		destroyWindow.Call(hwnd)
		return fmt.Errorf("failed to add clipboard format listener")
	}

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			postMessage.Call(hwnd, wmDestroy, 0, 0)
		case <-done:
		}
	}()

	var msg struct {
		Hwnd    syscall.Handle
		Message uint32
		WParam  uintptr
		LParam  uintptr
		Time    uint32
		Pt      struct{ X, Y int32 }
	}

	for {
		r, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		dispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}

	close(done)
	removeClipboardFormatListener.Call(hwnd)
	return nil
}

func (w *Clipboard) Write(p []byte) (n int, err error) {
	mimeType := http.DetectContentType(p)
	fmtType := FmtText
	if mimeType == "image/png" || mimeType == "image/jpeg" || mimeType == "image/gif" {
		fmtType = FmtImage
	}

	_, err = write(fmtType, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

var (
	errUnavailable = errors.New("clipboard unavailable")
	errUnsupported = errors.New("unsupported format")
)
