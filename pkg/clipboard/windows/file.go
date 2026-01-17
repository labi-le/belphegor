//go:build windows

package windows

import (
	"syscall"
	"unicode/utf16"
	"unsafe"
)

func (w *Clipboard) readFiles() ([]fileInfo, error) {
	hDrop, _, _ := syscall.SyscallN(getClipboardData.Addr(), cFmtHDrop)
	if hDrop == 0 {
		return nil, errUnavailable
	}

	count, _, _ := syscall.SyscallN(dragQueryFileW.Addr(), hDrop, 0xFFFFFFFF, 0, 0)
	if count == 0 {
		return nil, nil
	}

	limit := uintptr(w.opts.MaxClipboardFiles)
	if count < limit {
		limit = count
	}

	result := make([]fileInfo, 0, limit)
	var attr win32FileAttributeData

	for i := uintptr(0); i < limit; i++ {
		ln, _, _ := syscall.SyscallN(dragQueryFileW.Addr(), hDrop, i, 0, 0)
		if ln == 0 {
			continue
		}

		buf := make([]uint16, ln+1)
		res, _, _ := syscall.SyscallN(dragQueryFileW.Addr(), hDrop, i, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if res == 0 {
			continue
		}

		r1, _, _ := syscall.SyscallN(
			getFileAttributesEx.Addr(),
			uintptr(unsafe.Pointer(&buf[0])),
			getFileExInfoStandard,
			uintptr(unsafe.Pointer(&attr)),
		)

		info := fileInfo{
			Path: string(utf16.Decode(buf[:ln])),
		}

		if r1 != 0 {
			// skip folders
			if attr.FileAttributes&syscall.FILE_ATTRIBUTE_DIRECTORY != 0 {
				continue
			}

			info.Size = (uint64(attr.FileSizeHigh) << 32) | uint64(attr.FileSizeLow)
			info.ModTime = (uint64(attr.LastWriteTime.HighDateTime) << 32) | uint64(attr.LastWriteTime.LowDateTime)
		}

		result = append(result, info)
	}

	return result, nil
}
