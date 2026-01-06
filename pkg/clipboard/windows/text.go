//go:build windows

package windows

import (
	"fmt"
	"syscall"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"
)

func readText() ([]byte, error) {
	hMem, _, err := syscall.SyscallN(getClipboardData.Addr(), cFmtUnicodeText)
	if hMem == 0 {
		if err != 0 {
			return nil, err
		}
		return nil, nil
	}

	p, _, err := syscall.SyscallN(gLock.Addr(), hMem)
	if p == 0 {
		if err != 0 {
			return nil, err
		}
		return nil, fmt.Errorf("global lock failed")
	}
	defer noCheck(syscall.SyscallN(gUnlock.Addr(), hMem))

	// CF_UNICODETEXT: UTF-16LE, NUL-terminated
	u := (*uint16)(unsafe.Pointer(p))

	n := 0
	for {
		if *(*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(u)) + uintptr(n)*2)) == 0 {
			break
		}
		n++
	}

	s := unsafe.Slice(u, n)

	buf := make([]byte, 0, n)

	for i := 0; i < len(s); i++ {
		r := rune(s[i])

		if r < 128 {
			buf = append(buf, byte(r))
			continue
		}

		if 0xD800 <= r && r < 0xDC00 && i+1 < len(s) && 0xDC00 <= s[i+1] && s[i+1] < 0xE000 {
			r = utf16.DecodeRune(r, rune(s[i+1]))
			i++
			buf = utf8.AppendRune(buf, r)
			continue
		}

		if 0xD800 <= r && r < 0xE000 {
			r = utf8.RuneError
		}

		buf = utf8.AppendRune(buf, r)
	}

	return buf, nil
}

func writeText(buf []byte) error {
	r, _, err := syscall.SyscallN(emptyClipboard.Addr())
	if r == 0 {
		if err != 0 {
			return fmt.Errorf("failed to clear clipboard: %w", err)
		}
		return fmt.Errorf("failed to clear clipboard")
	}

	if len(buf) == 0 {
		return nil
	}

	len16 := 0
	for i := 0; i < len(buf); {
		r, size := utf8.DecodeRune(buf[i:])
		i += size
		if r >= 0x10000 {
			len16 += 2
			continue
		}
		len16++
	}
	len16++

	sizeBytes := uintptr(len16 * 2)

	hMem, _, err := syscall.SyscallN(gAlloc.Addr(), gmemMoveable, sizeBytes)
	if hMem == 0 {
		if err != 0 {
			return fmt.Errorf("failed to alloc global memory: %w", err)
		}
		return fmt.Errorf("failed to alloc global memory")
	}

	p, _, err := syscall.SyscallN(gLock.Addr(), hMem)
	if p == 0 {
		_, _, _ = syscall.SyscallN(gFree.Addr(), hMem)
		if err != 0 {
			return fmt.Errorf("failed to lock global memory: %w", err)
		}
		return fmt.Errorf("failed to lock global memory")
	}

	defer noCheck(syscall.SyscallN(gUnlock.Addr(), hMem))

	dst := unsafe.Slice((*uint16)(unsafe.Pointer(p)), len16)

	idx := 0
	for i := 0; i < len(buf); {
		r, size := utf8.DecodeRune(buf[i:])
		i += size

		if r >= 0x10000 {
			r1, r2 := utf16.EncodeRune(r)
			dst[idx] = uint16(r1)
			dst[idx+1] = uint16(r2)
			idx += 2
			continue
		}
		dst[idx] = uint16(r)
		idx++
	}
	dst[idx] = 0

	v, _, err := syscall.SyscallN(setClipboardData.Addr(), cFmtUnicodeText, hMem)
	if v == 0 {
		_, _, _ = syscall.SyscallN(gFree.Addr(), hMem)
		if err != 0 {
			return fmt.Errorf("failed to set text to clipboard: %w", err)
		}
		return fmt.Errorf("failed to set text to clipboard")
	}

	return nil
}
