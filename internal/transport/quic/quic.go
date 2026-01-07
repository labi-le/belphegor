package quic

import (
	"context"
	"crypto/tls"
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
		return nil, err
	}
	return &listenerAdapter{l}, nil
}

func (t *Transport) Dial(ctx context.Context, addr string) (transport.Connection, error) {
	conn, err := quic.DialAddr(ctx, addr, t.tlsConf, t.quicConf)
	if err != nil {
		return nil, err
	}
	return &connAdapter{conn: conn}, nil
}

type listenerAdapter struct {
	l *quic.Listener
}

func (a *listenerAdapter) Accept(ctx context.Context) (transport.Connection, error) {
	conn, err := a.l.Accept(ctx)
	if err != nil {
		return nil, err
	}
	return &connAdapter{conn: conn}, nil
}

func (a *listenerAdapter) Close() error   { return a.l.Close() }
func (a *listenerAdapter) Addr() net.Addr { return a.l.Addr() }

type connAdapter struct {
	conn *quic.Conn
}

func (c *connAdapter) OpenStream(ctx context.Context) (transport.Stream, error) {
	return c.conn.OpenStreamSync(ctx)
}

func (c *connAdapter) AcceptStream(ctx context.Context) (transport.Stream, error) {
	return c.conn.AcceptStream(ctx)
}

func (c *connAdapter) RemoteAddr() net.Addr { return c.conn.RemoteAddr() }

func (c *connAdapter) Close() error {
	return c.conn.CloseWithError(0, "closed")
}
