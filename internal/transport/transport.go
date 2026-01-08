package transport

import (
	"context"
	"errors"
	"io"
	"net"
	"time"
)

var (
	ErrStreamCanceled   = errors.New("stream canceled by remote peer")
	ErrConnectionClosed = errors.New("connection closed")
)

type Stream interface {
	io.Reader
	io.Writer
	io.Closer
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	Reset() error
}

type Connection interface {
	OpenStream(ctx context.Context) (Stream, error)
	AcceptStream(ctx context.Context) (Stream, error)

	RemoteAddr() net.Addr
	Close() error
}

type Listener interface {
	Accept(ctx context.Context) (Connection, error)
	Close() error
	Addr() net.Addr
}

type Transport interface {
	Listen(ctx context.Context, addr string) (Listener, error)
	Dial(ctx context.Context, addr string) (Connection, error)
}
