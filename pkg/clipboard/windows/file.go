//go:build windows

package windows

import (
	"syscall"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"

	"github.com/labi-le/belphegor/pkg/mime"
)

func readFileFirstMime() ([]byte, mime.Type, error) {
	det := mime.TypePath

	hDrop, _, _ := syscall.SyscallN(getClipboardData.Addr(), cFmtHDrop)
	if hDrop == 0 {
		return nil, det, errUnavailable
	}

	ln, _, _ := syscall.SyscallN(dragQueryFileW.Addr(), hDrop, 0, 0, 0)
	if ln == 0 {
		return nil, det, nil
	}

	buf := make([]uint16, ln+1)

	res, _, _ := syscall.SyscallN(dragQueryFileW.Addr(), hDrop, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if res == 0 {
		return nil, det, nil
	}

	out := make([]byte, 0, ln)

	slice := buf[:ln]

	for i := 0; i < len(slice); i++ {
		r := rune(slice[i])

		if r < 128 {
			out = append(out, byte(r))
			continue
		}

		if 0xD800 <= r && r < 0xDC00 && i+1 < len(slice) && 0xDC00 <= slice[i+1] && slice[i+1] < 0xE000 {
			r = utf16.DecodeRune(r, rune(slice[i+1]))
			i++
			out = utf8.AppendRune(out, r)
			continue
		}

		if 0xD800 <= r && r < 0xE000 {
			r = utf8.RuneError
		}
		out = utf8.AppendRune(out, r)
	}

	if len(out) == 0 {
		return nil, det, nil
	}

	return out, det, nil
}
