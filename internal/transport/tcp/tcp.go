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

// Transport implements transport.Transport over TLS-encrypted TCP
// with yamux for stream multiplexing.
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

func (t *Transport) Listen(_ context.Context, addr string) (transport.Listener, error) {
	l, err := tls.Listen("tcp", addr, t.tlsConf)
	if err != nil {
		return nil, err
	}
	return &listenerAdapter{
		l:         l,
		keepAlive: t.keepAlive,
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

// listenerAdapter wraps a TLS listener.
type listenerAdapter struct {
	l         net.Listener
	keepAlive time.Duration
}

func (a *listenerAdapter) Accept(ctx context.Context) (transport.Connection, error) {
	// net.Listener.Accept doesn't take context, so use a goroutine to
	// respect cancellation. The caller (multiListener.Close) owns the
	// listener lifecycle — we do NOT close the listener here on ctx cancel.
	type result struct {
		conn net.Conn
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		conn, err := a.l.Accept()
		ch <- result{conn, err}
	}()

	select {
	case <-ctx.Done():
		// Don't close the listener — let the owner (multiListener.Close) do it.
		// The goroutine will be unblocked when the listener is eventually closed.
		// Drain the result to avoid leaking the goroutine if a conn arrived.
		go func() {
			r := <-ch
			if r.conn != nil {
				_ = r.conn.Close()
			}
		}()
		return nil, ctx.Err()
	case r := <-ch:
		if r.err != nil {
			return nil, r.err
		}

		// Configure TCP keepalive on accepted connections (symmetric with Dial).
		if tlsConn, ok := r.conn.(*tls.Conn); ok {
			if tcpConn, ok := tlsConn.NetConn().(*net.TCPConn); ok {
				_ = tcpConn.SetKeepAlive(true)
				_ = tcpConn.SetKeepAlivePeriod(a.keepAlive)
			}
		}

		sess, err := yamux.Server(r.conn, yamuxConfig(a.keepAlive))
		if err != nil {
			_ = r.conn.Close()
			return nil, err
		}

		return &connAdapter{
			sess: sess,
		}, nil
	}
}

func (a *listenerAdapter) Close() error   { return a.l.Close() }
func (a *listenerAdapter) Addr() net.Addr { return a.l.Addr() }

// connAdapter wraps a yamux session over a TLS TCP connection.
// The yamux session owns the underlying net.Conn and closes it
// when the session is closed.
type connAdapter struct {
	sess *yamux.Session
}

func (c *connAdapter) OpenStream(ctx context.Context) (transport.Stream, error) {
	// yamux's OpenStream doesn't accept context. Pre-check is a best-effort
	// fast path; there is a TOCTOU window between this check and the call.
	// This is a known yamux limitation — it does not offer OpenStreamWithContext.
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

// Close shuts down the yamux session, which also closes the underlying connection.
func (c *connAdapter) Close() error {
	return c.sess.Close()
}

// streamAdapter wraps a yamux stream.
type streamAdapter struct {
	*yamux.Stream
}

func (s *streamAdapter) SetReadDeadline(t time.Time) error {
	return s.Stream.SetReadDeadline(t)
}

func (s *streamAdapter) SetWriteDeadline(t time.Time) error {
	return s.Stream.SetWriteDeadline(t)
}

// Reset aborts the stream. yamux v0.1.2 has no separate Reset API on *Stream —
// Close() is the only public method to terminate a stream. Unlike QUIC which
// has CancelRead/CancelWrite for abortive cancellation, yamux's stream model
// only supports graceful close.
func (s *streamAdapter) Reset() error {
	return s.Stream.Close()
}

func yamuxConfig(keepAlive time.Duration) *yamux.Config {
	cfg := yamux.DefaultConfig()
	cfg.EnableKeepAlive = true
	cfg.KeepAliveInterval = keepAlive
	cfg.ConnectionWriteTimeout = 10 * time.Second
	// Suppress noisy yamux logs
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
