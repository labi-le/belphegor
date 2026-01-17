package strutil

import (
	"unsafe"
)

func BytesToString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}
