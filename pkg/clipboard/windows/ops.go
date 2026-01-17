//go:build windows

package windows

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"time"

	"github.com/labi-le/belphegor/pkg/clipboard/eventful"
	"github.com/labi-le/belphegor/pkg/mime"
)

type capturedData struct {
	Bytes []byte
	Files []eventful.FileInfo
	Type  mime.Type
}

func (w *Clipboard) readDetected(t uintptr) (capturedData, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	closer, err := tryOpenClipboard()
	defer closer()
	if err != nil {
		return capturedData{}, fmt.Errorf("read detected: %w", err)
	}

	switch t {
	case cFmtDIBV5:
		b, err := w.readImage()
		if err != nil {
			return capturedData{}, fmt.Errorf("failed to read image: %w", err)
		}
		return capturedData{Bytes: b, Type: mime.TypeImage}, nil

	case cFmtHDrop:
		if !w.opts.AllowCopyFiles {
			return capturedData{}, fmt.Errorf("read detected file not allowed")
		}

		files, err := w.readFiles()
		if err != nil {
			return capturedData{}, err
		}

		return capturedData{Files: files, Type: mime.TypePath}, nil

	default:
		b, err := readText()
		if err != nil {
			return capturedData{}, fmt.Errorf("failed to read text: %w", err)
		}

		return capturedData{Bytes: b, Type: mime.TypeText}, nil
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
