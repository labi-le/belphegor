//go:build windows

package windows

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/cespare/xxhash"
	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/mime"
)

var _ eventful.Eventful = &Clipboard{}

var (
	errUnavailable = errors.New("clipboard unavailable")
	errUnsupported = errors.New("unsupported format")
)

const (
	debounce = 200 * time.Millisecond
)

var priorityList = []uint32{cFmtUnicodeText, cFmtDIBV5, cFmtHDrop}

func New() *Clipboard {
	return new(Clipboard)
}

type Clipboard struct {
	barrier  atomic.Int64
	lastHash atomic.Uint64
}

func (w *Clipboard) suppress() {
	deadline := time.Now().Add(debounce).UnixNano()
	w.barrier.Store(deadline)
}

func (w *Clipboard) allowed() bool {
	now := time.Now().UnixNano()
	deadline := w.barrier.Load()
	newDeadline := now + int64(debounce)

	if now < deadline {
		w.barrier.Store(newDeadline)
		return false
	}

	w.barrier.Store(newDeadline)
	return true
}

func (w *Clipboard) dedup(data []byte) bool {
	h := xxhash.Sum64(data)
	if h == w.lastHash.Load() {
		return false
	}
	w.lastHash.Store(h)
	return true
}

func (w *Clipboard) Watch(ctx context.Context, update chan<- eventful.Update) error {
	defer close(update)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hInstance, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	clsNamePtr, _ := syscall.UTF16PtrFromString(className)

	wndProc := syscall.NewCallback(func(hwnd syscall.Handle, msg uint32, wparam, lparam uintptr) uintptr {
		switch msg {
		case wmClipboardUpdate:
			if !w.allowed() {
				return 0
			}

			r, _, _ := syscall.SyscallN(getPriorityClipboardFormat.Addr(),
				uintptr(unsafe.Pointer(&priorityList[0])),
				uintptr(len(priorityList)),
			)

			if r == 0 {
				return 0
			}

			data, typ, err := readDetected(r)
			if err == nil {
				if !w.dedup(data) {
					return 0
				}

				update <- eventful.Update{
					Data:     data,
					MimeType: typ,
					Hash:     w.lastHash.Load(),
				}
			}
			return 0

		case wmDestroy:
			noCheck(syscall.SyscallN(postQuitMessage.Addr(), 0))
			return 0
		}

		ret, _, _ := syscall.SyscallN(defWindowProc.Addr(), uintptr(hwnd), uintptr(msg), wparam, lparam)
		return ret
	})

	wc := wndClassEx{
		Size:      uint32(unsafe.Sizeof(wndClassEx{})),
		Instance:  syscall.Handle(hInstance),
		WndProc:   wndProc,
		ClassName: clsNamePtr,
	}

	noCheck(syscall.SyscallN(registerClassEx.Addr(), uintptr(unsafe.Pointer(&wc))))

	hwnd, _, _ := syscall.SyscallN(createWindowEx.Addr(),
		0,
		uintptr(unsafe.Pointer(clsNamePtr)),
		uintptr(unsafe.Pointer(clsNamePtr)),
		0, 0, 0, 0, 0,
		0, 0, 0,
	)

	if hwnd == 0 {
		return fmt.Errorf("failed to create window listener")
	}

	ret, _, _ := syscall.SyscallN(addClipboardFormatListener.Addr(), hwnd)
	if ret == 0 {
		noCheck(syscall.SyscallN(destroyWindow.Addr(), hwnd))
		return fmt.Errorf("failed to add clipboard format listener")
	}

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			noCheck(syscall.SyscallN(postMessage.Addr(), hwnd, wmDestroy, 0, 0))
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
		r, _, _ := syscall.SyscallN(getMessage.Addr(), uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		noCheck(syscall.SyscallN(translateMessage.Addr(), uintptr(unsafe.Pointer(&msg))))
		noCheck(syscall.SyscallN(dispatchMessage.Addr(), uintptr(unsafe.Pointer(&msg))))
	}

	close(done)
	noCheck(syscall.SyscallN(removeClipboardFormatListener.Addr(), hwnd))
	return nil
}

func (w *Clipboard) Write(p []byte) (n int, err error) {
	w.suppress()

	if err := write(p, mime.From(p)); err != nil {
		return 0, err
	}

	return len(p), nil
}
