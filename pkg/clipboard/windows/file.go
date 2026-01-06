//go:build windows

package windows

import (
	"syscall"
	"unsafe"

	"github.com/labi-le/belphegor/pkg/mime"
)

func readFileFirstMime() ([]byte, mime.Type, error) {
	det := mime.TypePath

	hDrop, _, _ := getClipboardData.Call(cFmtHDrop)
	if hDrop == 0 {
		return nil, det, errUnavailable
	}

	ln, _, _ := dragQueryFileW.Call(hDrop, 0, 0, 0)
	if ln == 0 {
		return nil, det, nil
	}

	buf := make([]uint16, ln+1)
	_, _, _ = dragQueryFileW.Call(hDrop, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))

	path := syscall.UTF16ToString(buf)
	if path == "" {
		return nil, det, nil
	}

	return []byte(path), det, nil
}
