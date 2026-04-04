package auto

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/labi-le/belphegor/internal/transport"
	"github.com/labi-le/belphegor/internal/transport/quic"
	"github.com/labi-le/belphegor/internal/transport/tcp"
	"github.com/rs/zerolog"
)

const (
	// dialProbeTimeout is the maximum time to wait for a QUIC dial before
	// falling back to TCP.
	dialProbeTimeout = 5 * time.Second
)

// Transport implements transport.Transport with automatic protocol selection.
// It listens on both QUIC (UDP) and TCP simultaneously, and when dialing it
// tries QUIC first with a short timeout, falling back to TCP.
type Transport struct {
	quicTransport *quic.Transport
	tcpTransport  *tcp.Transport
	logger        zerolog.Logger
}

// New creates a new auto-selecting Transport that prefers QUIC but falls back
// to TCP when QUIC is unavailable (e.g. behind restrictive firewalls or relay networks).
func New(tlsConf *tls.Config, keepAlive time.Duration, logger zerolog.Logger) *Transport {
	return &Transport{
		quicTransport: quic.New(tlsConf, keepAlive),
		tcpTransport:  tcp.New(tlsConf, keepAlive),
		logger:        logger,
	}
}

var _ transport.Transport = (*Transport)(nil)

// Listen starts both a QUIC and TCP listener on the same port.
// The returned multiListener accepts connections from either transport.
func (t *Transport) Listen(ctx context.Context, addr string) (transport.Listener, error) {
	quicL, quicErr := t.quicTransport.Listen(ctx, addr)
	tcpL, tcpErr := t.tcpTransport.Listen(ctx, addr)

	if quicErr != nil && tcpErr != nil {
		return nil, fmt.Errorf("auto: both listeners failed: %w", errors.Join(quicErr, tcpErr))
	}

	if quicErr != nil {
		t.logger.Warn().Err(quicErr).Msg("auto: QUIC listener failed, TCP only")
	}
	if tcpErr != nil {
		t.logger.Warn().Err(tcpErr).Msg("auto: TCP listener failed, QUIC only")
	}

	return newMultiListener(quicL, tcpL, t.logger), nil
}

// Dial tries QUIC first with a short timeout. If QUIC fails (common behind
// restrictive firewalls or DERP relays), it falls back to TCP.
func (t *Transport) Dial(ctx context.Context, addr string) (transport.Connection, error) {
	probeCtx, cancel := context.WithTimeout(ctx, dialProbeTimeout)
	defer cancel()

	conn, err := t.quicTransport.Dial(probeCtx, addr)
	if err == nil {
		t.logger.Debug().Str("addr", addr).Msg("auto: connected via QUIC")
		return conn, nil
	}

	t.logger.Debug().Err(err).Str("addr", addr).Msg("auto: QUIC failed, trying TCP")

	conn, err = t.tcpTransport.Dial(ctx, addr)
	if err != nil {
		return nil, fmt.Errorf("auto: all transports failed for %s: %w", addr, err)
	}

	t.logger.Debug().Str("addr", addr).Msg("auto: connected via TCP")
	return conn, nil
}

// acceptResult is the result of an Accept call from a listener.
type acceptResult struct {
	conn transport.Connection
	err  error
}

// multiListener accepts connections from both QUIC and TCP listeners using
// persistent accept goroutines to avoid goroutine leaks.
type multiListener struct {
	quicL       transport.Listener
	tcpL        transport.Listener
	connCh      chan acceptResult
	cancel      context.CancelFunc
	activeCount atomic.Int32
	logger      zerolog.Logger
}

// newMultiListener creates a multiListener and starts persistent accept loops.
func newMultiListener(quicL, tcpL transport.Listener, logger zerolog.Logger) *multiListener {
	ctx, cancel := context.WithCancel(context.Background())
	m := &multiListener{
		quicL:  quicL,
		tcpL:   tcpL,
		connCh: make(chan acceptResult, 1),
		cancel: cancel,
		logger: logger,
	}
	if quicL != nil {
		m.activeCount.Add(1)
		go m.acceptLoop(ctx, quicL, "quic")
	}
	if tcpL != nil {
		m.activeCount.Add(1)
		go m.acceptLoop(ctx, tcpL, "tcp")
	}
	return m
}

// acceptLoop continuously accepts connections from a single listener and
// sends them to the shared channel. It exits when the context is canceled
// or the listener returns a fatal error.
//
// When a successful connection is received, it is always delivered to connCh
// before checking ctx — this prevents silently dropping valid connections
// when cancellation races with a successful accept.
//
// When this loop exits due to an error, it decrements activeCount. Only the
// last loop to exit forwards the error to connCh — if the other transport is
// still healthy we log a warning and let it keep accepting.
func (m *multiListener) acceptLoop(ctx context.Context, l transport.Listener, name string) {
	for {
		conn, err := l.Accept(ctx)

		if err != nil {
			// This listener is dead. Decide whether to surface the error
			// BEFORE writing to connCh, so we never forward a per-transport
			// error when the other transport is still healthy.
			remaining := m.activeCount.Add(-1)
			if remaining > 0 {
				// At least one other transport is still alive — log and exit
				// quietly so node.Start keeps accepting from the survivor.
				m.logger.Warn().Err(err).Str("transport", name).
					Msg("auto: listener failed, continuing on remaining transport(s)")
				return
			}
			// Last listener gone — signal Accept() that the multiListener is done.
			select {
			case m.connCh <- acceptResult{nil, fmt.Errorf("auto: all listeners failed: %w", err)}:
			case <-ctx.Done():
			}
			return
		}

		// No error: deliver the connection. If ctx is already canceled, close
		// the connection and exit rather than block forever.
		select {
		case m.connCh <- acceptResult{conn, nil}:
		case <-ctx.Done():
			_ = conn.Close()
			return
		}
	}
}

// Accept returns the next connection from either listener.
func (m *multiListener) Accept(ctx context.Context) (transport.Connection, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-m.connCh:
		return r.conn, r.err
	}
}

// Close shuts down both listeners and cancels the accept loops.
func (m *multiListener) Close() error {
	m.cancel()
	var errs []error
	if m.quicL != nil {
		errs = append(errs, m.quicL.Close())
	}
	if m.tcpL != nil {
		errs = append(errs, m.tcpL.Close())
	}
	return errors.Join(errs...)
}

// Addr returns the address of the first available listener.
func (m *multiListener) Addr() net.Addr {
	if m.quicL != nil {
		return m.quicL.Addr()
	}
	if m.tcpL != nil {
		return m.tcpL.Addr()
	}
	return nil
}
