package quic

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"time"

	"github.com/labi-le/belphegor/internal/transport"
	"github.com/quic-go/quic-go"
)

type Transport struct {
	tlsConf  *tls.Config
	quicConf *quic.Config
}

func New(tlsConf *tls.Config, keepAlive time.Duration) *Transport {
	return &Transport{
		tlsConf: tlsConf,
		quicConf: &quic.Config{
			KeepAlivePeriod: keepAlive,
		},
	}
}

var _ transport.Transport = (*Transport)(nil)

func (t *Transport) Listen(_ context.Context, addr string) (transport.Listener, error) {
	l, err := quic.ListenAddr(addr, t.tlsConf, t.quicConf)
	if err != nil {
		return nil, mapQuicError(err)
	}
	return &listenerAdapter{l}, nil
}

func (t *Transport) Dial(ctx context.Context, addr string) (transport.Connection, error) {
	conn, err := quic.DialAddr(ctx, addr, t.tlsConf, t.quicConf)
	if err != nil {
		return nil, mapQuicError(err)
	}
	return &connAdapter{conn: conn}, nil
}

type listenerAdapter struct {
	l *quic.Listener
}

func (a *listenerAdapter) Accept(ctx context.Context) (transport.Connection, error) {
	conn, err := a.l.Accept(ctx)
	if err != nil {
		return nil, mapQuicError(err)
	}
	return &connAdapter{conn: conn}, nil
}

func (a *listenerAdapter) Close() error   { return mapQuicError(a.l.Close()) }
func (a *listenerAdapter) Addr() net.Addr { return a.l.Addr() }

type connAdapter struct {
	conn *quic.Conn
}

func (c *connAdapter) OpenStream(ctx context.Context) (transport.Stream, error) {
	s, err := c.conn.OpenStreamSync(ctx)
	if err != nil {
		return nil, mapQuicError(err)
	}
	return streamAdapter{s}, nil
}

func (c *connAdapter) AcceptStream(ctx context.Context) (transport.Stream, error) {
	s, err := c.conn.AcceptStream(ctx)
	if err != nil {
		return nil, mapQuicError(err)
	}
	return streamAdapter{s}, nil
}

func (c *connAdapter) RemoteAddr() net.Addr { return c.conn.RemoteAddr() }

func (c *connAdapter) Close() error {
	return c.conn.CloseWithError(0, "closed")
}

type streamAdapter struct {
	*quic.Stream
}

func (s streamAdapter) Read(p []byte) (n int, err error) {
	n, err = s.Stream.Read(p)
	return n, mapQuicError(err)
}

func (s streamAdapter) Write(p []byte) (n int, err error) {
	n, err = s.Stream.Write(p)
	return n, mapQuicError(err)
}

func (s streamAdapter) Reset() error {
	s.CancelRead(stopSend)
	s.CancelWrite(stopSend)
	return nil
}

const (
	stopSend = 0
)

func mapQuicError(err error) error {
	if err == nil {
		return nil
	}

	var streamErr *quic.StreamError
	if errors.As(err, &streamErr) {
		if streamErr.ErrorCode == stopSend {
			return transport.ErrStreamCanceled
		}
	}

	return err
}
