//go:build windows

package windows

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"time"

	"github.com/labi-le/belphegor/pkg/mime"
)

func readDetected(t uintptr) ([]byte, mime.Type, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	closer, err := tryOpenClipboard()
	defer closer()
	if err != nil {
		return nil, mime.TypeUnknown, fmt.Errorf("read detected: %w", err)
	}

	switch t {
	case cFmtDIBV5:
		b, err := readImage()
		if err != nil {
			return nil, mime.TypeUnknown, fmt.Errorf("failed to read image: %w", err)
		}
		return b, mime.TypeImage, nil

	case cFmtHDrop:
		// todo: in future possible return multiply paths
		return readFileFirstMime()

	default:
		b, err := readText()
		if err != nil {
			return nil, mime.TypeUnknown, fmt.Errorf("failed to read text: %w", err)
		}
		return b, mime.TypeText, nil
	}
}

func tryOpenClipboard() (func(), error) {
	for i := 0; i < 5; i++ {
		r, _, _ := syscall.SyscallN(openClipboard.Addr(), 0)
		if r != 0 {
			return func() { noCheck(syscall.SyscallN(closeClipboard.Addr())) }, nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return func() {}, errors.New("failed to open clipboard")
}

func write(typ mime.Type, buf []byte) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	closer, err := tryOpenClipboard()
	defer closer()
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	switch typ {
	case mime.TypeImage:
		if err := writeImage(buf); err != nil {
			return fmt.Errorf("failed to write image: %w", err)
		}
	case mime.TypeText:
		fallthrough
	default:
		if err := writeText(buf); err != nil {
			return fmt.Errorf("failed to write text: %w", err)
		}
	}

	return nil
}

func noCheck(_ uintptr, _ uintptr, _ syscall.Errno) {}
