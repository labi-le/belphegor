package pipe

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/labi-le/belphegor/pkg/pool/byteslice"
	"github.com/rs/zerolog"
	"golang.org/x/sys/unix"
)

var _ RWPipe = &Reusable{}

var (
	ErrNilPipe      = fmt.Errorf("pipe: nil pipe provided")
	ErrFailedCreate = fmt.Errorf("pipe: failed to create pipe")
	ErrClose        = fmt.Errorf("pipe: failted to close")
)

type RWPipe interface {
	// Fd returns a valid file descriptor for write
	Fd() *os.File
	// ReadFd returns a valid file descriptor for read
	ReadFd() *os.File
	// Close pipe
	Close() error
}

type Fd interface {
	Fd() uintptr
	Close() error
}

// Reusable
//
// Non thread-safe
type Reusable struct {
	rfd, wfd *os.File
	logger   zerolog.Logger
	mu       sync.RWMutex
}

func MustNonBlock(log zerolog.Logger) *Reusable {
	p, err := NewNonBlock(log)
	if err != nil {
		panic(err)
	}

	return p
}

func NewNonBlock(log zerolog.Logger) (*Reusable, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return &Reusable{}, errors.Join(ErrFailedCreate, err)
	}

	return &Reusable{
		rfd:    r,
		wfd:    w,
		logger: log.With().Str("component", "pipe").Logger(),
	}, nil
}

func (w *Reusable) Close() error {
	w.logger.Trace().Msg("close called")

	//w.mu.Lock()
	//defer w.mu.Unlock()

	w.logger.Trace().Ints("close(:rfd, :wfd)", []int{int(w.rfd.Fd()), int(w.wfd.Fd())}).Send()

	if err := w.rfd.Close(); err != nil {
		return errors.Join(ErrClose, err)
	}
	if err := w.wfd.Close(); err != nil {
		return errors.Join(ErrClose, err)
	}

	return nil
}

func (w *Reusable) Fd() *os.File {
	w.logger.Trace().Msg("fd called")

	w.mu.Lock()
	defer w.mu.Unlock()

	w.logger.Trace().Int("write_fd", int(w.wfd.Fd())).Send()
	return w.wfd
}

func (w *Reusable) ReadFd() *os.File {
	w.logger.Trace().Msg("readfd called")

	w.mu.Lock()
	defer w.mu.Unlock()

	w.logger.Trace().Int("read_fd", int(w.rfd.Fd())).Send()
	return w.rfd
}

func waitForReadable(fd uintptr, lastRead time.Time, hasData bool) (int, bool, error) {
	if hasData && time.Since(lastRead) >= 5*time.Millisecond {
		return 0, true, nil
	}

	pfd := unix.PollFd{
		Fd:     int32(fd),
		Events: unix.POLLIN | unix.POLLPRI | unix.POLLERR | unix.POLLHUP | unix.POLLNVAL,
	}

	pollTimeout := uintptr(100) // 100ms timeout after read first portion cake
	if !hasData {
		pollTimeout = ^uintptr(0) // inf wait
	}

	for {
		_, _, errno := syscall.Syscall(syscall.SYS_POLL,
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

		if pfd.Revents&(unix.POLLERR|unix.POLLNVAL|unix.POLLHUP) != 0 {
			return 0, false, fmt.Errorf("poll error event: %v", pfd.Revents)
		}

		if pfd.Revents&unix.POLLIN != 0 {
			return readableSize(fd), false, nil
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
