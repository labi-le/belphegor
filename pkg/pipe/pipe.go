package pipe

import (
	"errors"
	"fmt"
	"github.com/labi-le/belphegor/pkg/pool/byteslice"
	"golang.org/x/sys/unix"
	"syscall"
	"time"
	"unsafe"
)

var _ RWPipe = Reusable{}

var (
	ErrNilPipe      = fmt.Errorf("pipe: nil pipe provided")
	ErrFailedCreate = fmt.Errorf("pipe: failed to create pipe")
	ErrClose        = fmt.Errorf("pipe: failted to close")
)

type RWPipe interface {
	// Fd returns a valid file descriptor for write
	Fd() uintptr
	// ReadFd returns a valid file descriptor for read
	ReadFd() uintptr
	// Close pipe
	Close() error
}

// Reusable
//
// The “Way” of this samurai is to process data endlessly.
// The main point is to get meaningful data while remaining unblockable.
// Non thread-safe
type Reusable struct {
	rfd, wfd int
}

func MustNonBlock() Reusable {
	p, err := NewNonBlock()
	if err != nil {
		panic(err)
	}

	return p
}

func NewNonBlock() (Reusable, error) {
	var pipefd [2]int
	if err := syscall.Pipe(pipefd[:]); err != nil {
		return Reusable{}, errors.Join(ErrFailedCreate, err)
	}

	if err := syscall.SetNonblock(pipefd[0], true); err != nil {
		return Reusable{}, errors.Join(ErrFailedCreate, err)
	}

	return Reusable{
		rfd: pipefd[0],
		wfd: pipefd[1],
	}, nil
}

func (w Reusable) Close() error {
	if err := syscall.Close(w.rfd); err != nil {
		return errors.Join(ErrClose, err)
	}
	if err := syscall.Close(w.wfd); err != nil {
		return errors.Join(ErrClose, err)
	}
	return nil
}

func (w Reusable) Fd() uintptr {
	return uintptr(w.wfd)
}

func (w Reusable) ReadFd() uintptr {
	return uintptr(w.rfd)
}

func waitForReadable(fd uintptr, lastRead time.Time, hasData bool) (int, bool, error) {
	pfd := unix.PollFd{
		Fd:     int32(fd),
		Events: unix.POLLIN | unix.POLLPRI | unix.POLLERR | unix.POLLHUP | unix.POLLNVAL,
	}

	const (
		pollTimeout = 5 // ms
		maxIdleTime = 5 * time.Millisecond
	)

	// checking whether the timeout between messages has expired
	if hasData && time.Since(lastRead) >= maxIdleTime {
		return 0, true, nil // timeout
	}

	for {
		n, _, errno := syscall.Syscall(syscall.SYS_POLL,
			uintptr(unsafe.Pointer(&pfd)),
			1,
			pollTimeout,
		)

		if errno != 0 {
			if errno == syscall.EINTR {
				continue
			}
			return 0, false, fmt.Errorf("poll error: %w", errno)
		}

		if n == 0 {
			return 0, false, nil // no data, but this is not the final timeout
		}

		if pfd.Revents&(unix.POLLERR|unix.POLLNVAL|unix.POLLHUP) != 0 {
			return 0, false, fmt.Errorf("poll error event: %v", pfd.Revents)
		}

		if pfd.Revents&unix.POLLIN != 0 {
			return size(fd), false, nil
		}

		return 0, false, nil
	}
}
func FromPipe(pipe uintptr) ([]byte, error) {
	if pipe == 0 {
		return nil, ErrNilPipe
	}

	buffer := byteslice.Get(capacity(pipe))
	defer byteslice.Put(buffer)
	total := 0

	lastRead := time.Now()
	for {
		size, timeout, err := waitForReadable(pipe, lastRead, total > 0)
		if err != nil {
			return nil, err
		}

		if timeout {
			break
		}

		if size == 0 {
			continue
		}

		if total+size > cap(buffer) {
			newCap := cap(buffer) * 2
			if newCap < total+size {
				newCap = total + size
			}

			newBuf := byteslice.Get(newCap)
			copy(newBuf, buffer[:total])
			byteslice.Put(buffer)
			buffer = newBuf
		}

		n, err := syscall.Read(int(pipe), buffer[total:cap(buffer)])
		if err != nil {
			if errCode, ok := err.(syscall.Errno); ok &&
				(errCode == syscall.EAGAIN || errCode == syscall.EINTR) {
				continue
			}
			return nil, err
		}

		if n > 0 {
			total += n
			lastRead = time.Now()
		}
	}

	return buffer[:total], nil
}
