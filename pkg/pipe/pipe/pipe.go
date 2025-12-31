//go:build unix

package pipe

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

var (
	ErrNilPipe      = fmt.Errorf("pipe: nil pipe provided")
	ErrFailedCreate = fmt.Errorf("pipe: failed to create pipe")
)

type RWPipe interface {
	// Fd returns write file descriptor for Wayland (ownership will be transferred)
	Fd() *os.File
	// ReadFd returns read file descriptor
	ReadFd() *os.File
	// Close closes only the read end
	Close() error
}

// Reusable represents a pipe for reading clipboard data from Wayland
type Reusable struct {
	rfd *os.File
	wfd *os.File
}

func New() (*Reusable, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return nil, errors.Join(ErrFailedCreate, err)
	}

	return &Reusable{
		rfd: r,
		wfd: w,
	}, nil
}

// Close closes only the read end. Write end is managed by Wayland compositor
func (p *Reusable) Close() error {
	return p.rfd.Close()
}

func (p *Reusable) Fd() *os.File {
	return p.wfd
}

func (p *Reusable) ReadFd() *os.File {
	return p.rfd
}

func FromPipe(pipe uintptr) ([]byte, error) {
	if pipe == 0 {
		return nil, ErrNilPipe
	}

	const (
		readChunkSize = 64 * 1024
		readTimeout   = 100 * time.Millisecond
		dataDelay     = 10 * time.Millisecond
	)

	var dest bytes.Buffer

	readBuf := make([]byte, readChunkSize)

	lastRead := time.Now()
	hasData := false

	for {
		timeout, err := waitForData(pipe, lastRead, hasData, readTimeout, dataDelay)
		if err != nil {
			return nil, err
		}

		if timeout {
			break
		}

		n, err := syscall.Read(int(pipe), readBuf)
		if err != nil && !needWait(err) {
			return nil, err
		}

		if n == 0 {
			break
		}

		if n > 0 {
			dest.Write(readBuf[:n])
		}

		lastRead = time.Now()
		hasData = true
	}

	return dest.Bytes(), nil
}

func needWait(err error) bool {
	var errno syscall.Errno
	return errors.As(err, &errno) && (errors.Is(errno, syscall.EAGAIN) || errors.Is(errno, syscall.EINTR))
}

func waitForData(fd uintptr, lastRead time.Time, hasData bool, readTimeout, dataDelay time.Duration) (bool, error) {
	if hasData && time.Since(lastRead) >= dataDelay {
		return true, nil
	}

	fds := []unix.PollFd{{
		Fd:     int32(fd),
		Events: unix.POLLIN | unix.POLLERR | unix.POLLHUP | unix.POLLNVAL,
	}}

	timeout := -1
	if hasData {
		timeout = int(readTimeout.Milliseconds())
	}

	for {
		n, err := unix.Poll(fds, timeout)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			return false, fmt.Errorf("poll error: %w", err)
		}

		if n == 0 {
			return true, nil
		}

		re := fds[0].Revents
		if re&(unix.POLLERR|unix.POLLNVAL) != 0 {
			return true, fmt.Errorf("poll error revents=%v", re)
		}
		if re&unix.POLLHUP != 0 {
			return true, nil
		}
		if re&unix.POLLIN != 0 {
			return false, nil
		}
	}
}
