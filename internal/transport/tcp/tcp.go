package tcp

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/labi-le/belphegor/internal/transport"
)

type Transport struct {
	tlsConf   *tls.Config
	keepAlive time.Duration
}

func New(tlsConf *tls.Config, keepAlive time.Duration) *Transport {
	return &Transport{
		tlsConf:   tlsConf,
		keepAlive: keepAlive,
	}
}

var _ transport.Transport = (*Transport)(nil)

func (t *Transport) Listen(ctx context.Context, addr string) (transport.Listener, error) {
	lc := net.ListenConfig{}
	rawListener, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}

	l := tls.NewListener(rawListener, t.tlsConf)

	ctx, cancel := context.WithCancel(ctx)
	go func() {
		<-ctx.Done()
		_ = l.Close()
	}()

	return &listenerAdapter{
		l:         l,
		keepAlive: t.keepAlive,
		cancel:    cancel,
	}, nil
}

func (t *Transport) Dial(ctx context.Context, addr string) (transport.Connection, error) {
	dialer := &tls.Dialer{Config: t.tlsConf}
	rawConn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}

	tcpConn := rawConn.(*tls.Conn).NetConn()
	if tc, ok := tcpConn.(*net.TCPConn); ok {
		_ = tc.SetKeepAlive(true)
		_ = tc.SetKeepAlivePeriod(t.keepAlive)
	}

	sess, err := yamux.Client(rawConn, yamuxConfig(t.keepAlive))
	if err != nil {
		_ = rawConn.Close()
		return nil, err
	}

	return &connAdapter{
		sess: sess,
	}, nil
}

type listenerAdapter struct {
	l         net.Listener
	keepAlive time.Duration
	cancel    context.CancelFunc
}

func (a *listenerAdapter) Accept(_ context.Context) (transport.Connection, error) {
	conn, err := a.l.Accept()
	if err != nil {
		return nil, err
	}

	if tlsConn, ok := conn.(*tls.Conn); ok {
		if tcpConn, ok := tlsConn.NetConn().(*net.TCPConn); ok {
			_ = tcpConn.SetKeepAlive(true)
			_ = tcpConn.SetKeepAlivePeriod(a.keepAlive)
		}
	}

	sess, err := yamux.Server(conn, yamuxConfig(a.keepAlive))
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	return &connAdapter{
		sess: sess,
	}, nil
}

func (a *listenerAdapter) Close() error {
	a.cancel()
	return a.l.Close()
}

func (a *listenerAdapter) Addr() net.Addr { return a.l.Addr() }

type connAdapter struct {
	sess *yamux.Session
}

func (c *connAdapter) OpenStream(ctx context.Context) (transport.Stream, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s, err := c.sess.OpenStream()
	if err != nil {
		return nil, mapError(err)
	}
	return &streamAdapter{s}, nil
}

func (c *connAdapter) AcceptStream(ctx context.Context) (transport.Stream, error) {
	s, err := c.sess.AcceptStreamWithContext(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	return &streamAdapter{s}, nil
}

func (c *connAdapter) RemoteAddr() net.Addr { return c.sess.RemoteAddr() }

func (c *connAdapter) Close() error {
	return c.sess.Close()
}

type streamAdapter struct {
	*yamux.Stream
}

func (s *streamAdapter) SetReadDeadline(t time.Time) error {
	return s.Stream.SetReadDeadline(t)
}

func (s *streamAdapter) SetWriteDeadline(t time.Time) error {
	return s.Stream.SetWriteDeadline(t)
}

func (s *streamAdapter) Reset() error {
	return s.Stream.Close()
}

func yamuxConfig(keepAlive time.Duration) *yamux.Config {
	cfg := yamux.DefaultConfig()
	cfg.EnableKeepAlive = true
	cfg.KeepAliveInterval = keepAlive
	cfg.LogOutput = io.Discard
	return cfg
}

func mapError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, yamux.ErrSessionShutdown) ||
		errors.Is(err, yamux.ErrStreamsExhausted) ||
		errors.Is(err, yamux.ErrConnectionReset) {
		return transport.ErrConnectionClosed
	}

	if errors.Is(err, yamux.ErrStreamClosed) {
		return transport.ErrStreamCanceled
	}

	return err
}
