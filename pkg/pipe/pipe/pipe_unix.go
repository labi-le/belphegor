package pipe

import (
	"golang.org/x/sys/unix"
	"syscall"
	"unsafe"
)

func readableSize(fd uintptr) int {
	var length int
	syscall.Syscall(
		syscall.SYS_IOCTL,
		fd,
		syscall.TIOCINQ,
		uintptr(unsafe.Pointer(&length)),
	)

	return length
}

func increaseSize(fd uintptr, size int) int {
	fcntlInt, err := unix.FcntlInt(fd, syscall.F_SETPIPE_SZ, size)
	if err != nil {
		panic(err)
	}
	return fcntlInt
}

// capacity returns the total capacity of the pipe
func capacity(fd uintptr) int {
	fcntlInt, err := unix.FcntlInt(fd, syscall.F_GETPIPE_SZ, 0)
	if err != nil {
		return 0
	}
	return fcntlInt
}
