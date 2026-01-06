//go:build windows

package windows

import (
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/labi-le/belphegor/pkg/mime"
)

type format int

const (
	fmtText format = iota
	fmtImage
	fmtFile
)

func readDetected(t format) ([]byte, mime.Type, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var format uintptr
	switch t {
	case fmtImage:
		format = cFmtDIBV5
	case fmtFile:
		format = cFmtHDrop
	case fmtText:
		fallthrough
	default:
		format = cFmtUnicodeText
	}

	closer, err := tryOpenClipboard()
	defer closer()
	if err != nil {
		return nil, mime.TypeUnknown, fmt.Errorf("read detected: %w", err)
	}

	switch format {
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
		r, _, _ := openClipboard.Call(0)
		if r != 0 {
			return func() { _, _, _ = closeClipboard.Call() }, nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return func() {}, errors.New("failed to open clipboard")
}

func write(t format, buf []byte) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	closer, err := tryOpenClipboard()
	defer closer()
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	switch t {
	case fmtImage:
		if err := writeImage(buf); err != nil {
			return fmt.Errorf("failed to write image: %w", err)
		}
	case fmtText:
		fallthrough
	default:
		if err := writeText(buf); err != nil {
			return fmt.Errorf("failed to write text: %w", err)
		}
	}

	return nil
}
