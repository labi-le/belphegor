//go:build windows

package windows

import (
	"errors"
	"runtime"
	"time"

	"github.com/labi-le/belphegor/pkg/mime"
)

type Format int

const (
	FmtText Format = iota
	FmtImage
	FmtFile
)

func ReadDetected(t Format) ([]byte, mime.Type) {
	buf, det, err := readDetected(t)
	if err != nil {
		return nil, mime.TypeUnknown
	}
	return buf, det
}

func readDetected(t Format) (buf []byte, det mime.Type, err error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	det = mime.TypeUnknown

	var format uintptr
	switch t {
	case FmtImage:
		format = cFmtDIBV5
	case FmtFile:
		format = cFmtHDrop
	case FmtText:
		fallthrough
	default:
		format = cFmtUnicodeText
	}

	r, _, err := isClipboardFormatAvailable.Call(format)
	if r == 0 {
		return nil, det, errUnavailable
	}

	for i := 0; i < 5; i++ {
		r, _, _ = openClipboard.Call(0)
		if r != 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if r == 0 {
		return nil, det, errors.New("failed to open clipboard")
	}
	defer closeClipboard.Call()

	switch format {
	case cFmtDIBV5:
		b, err := readImage()
		if err != nil {
			return nil, det, err
		}
		det = mime.TypeImage
		return b, det, nil

	case cFmtHDrop:
		return readFileFirstMime()

	default:
		b, err := readText()
		if err != nil {
			return nil, det, err
		}
		det = mime.TypeText
		return b, det, nil
	}
}

func write(t Format, buf []byte) (<-chan struct{}, error) {
	errch := make(chan error)
	changed := make(chan struct{}, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		// Retry loop for OpenClipboard
		var r uintptr
		for i := 0; i < 5; i++ {
			r, _, _ = openClipboard.Call(0)
			if r != 0 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if r == 0 {
			errch <- errors.New("failed to open clipboard")
			return
		}

		switch t {
		case FmtImage:
			err := writeImage(buf)
			if err != nil {
				errch <- err
				closeClipboard.Call()
				return
			}
		case FmtText:
			fallthrough
		default:
			err := writeText(buf)
			if err != nil {
				errch <- err
				closeClipboard.Call()
				return
			}
		}
		closeClipboard.Call()

		errch <- nil
		close(changed)
	}()

	err := <-errch
	if err != nil {
		return nil, err
	}
	return changed, nil
}
