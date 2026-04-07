package peer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/labi-le/belphegor/internal/channel"
	"github.com/labi-le/belphegor/internal/protocol"
	"github.com/labi-le/belphegor/internal/store"
	"github.com/labi-le/belphegor/internal/transport"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/ctxlog"
	"github.com/labi-le/belphegor/pkg/network"
	"github.com/rs/zerolog"
)

type Options struct {
	Channel        *channel.Channel
	Store          store.FileWriter
	Logger         zerolog.Logger
	Deadline       network.Deadline
	MaxReceiveSize uint64
	Batches        *channel.BatchCollector
}

type Peer struct {
	conn       transport.Connection
	metaData   domain.Device
	channel    *channel.Channel
	stringRepr string
	logger     zerolog.Logger
	deadline   network.Deadline

	fileWriter     store.FileWriter
	maxReceiveSize uint64
	batches        *channel.BatchCollector
}

func New(
	conn transport.Connection,
	metadata domain.Device,
	opts Options,
) *Peer {
	return &Peer{
		conn:           conn,
		metaData:       metadata,
		channel:        opts.Channel,
		fileWriter:     opts.Store,
		logger:         opts.Logger,
		deadline:       opts.Deadline,
		stringRepr:     fmt.Sprintf("%s -> %s", metadata.Name, conn.RemoteAddr().String()),
		maxReceiveSize: opts.MaxReceiveSize,
		batches:        opts.Batches,
	}
}

func (p *Peer) MetaData() domain.Device { return p.metaData }

func (p *Peer) Conn() transport.Connection { return p.conn }

func (p *Peer) Close() error {
	return p.conn.Close()
}

func (p *Peer) String() string {
	return p.stringRepr
}

func (p *Peer) Receive(ctx context.Context) error {
	ctxLog := ctxlog.Op(p.logger, "peer.Receive")
	defer ctxLog.
		Info().
		Str("node", p.String()).
		Msg("disconnected")

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			stream, err := p.conn.AcceptStream(ctx)
			if err != nil {
				if isConnClosed(err) {
					return nil
				}
				ctxLog.Info().Err(err).Msg("failed to accept stream, closing connection")
				return fmt.Errorf("peer.Receive: %w", err)
			}

			go func() {
				if handleErr := p.handleStream(ctx, stream); handleErr != nil {
					ctxLog.Trace().Err(handleErr).Msg("failed to handle stream")
				}
			}()
		}
	}
}

func isConnClosed(err error) bool {
	if errors.Is(err, context.Canceled) {
		return true
	}

	if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
		return true
	}

	msg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(msg, "closed"):
		return true
	case strings.Contains(msg, "application error 0x0"):
		return true
	case strings.Contains(msg, "unreachable"):
		return true
	case strings.Contains(msg, "reset"):
		return true
	case strings.Contains(msg, "broken pipe"):
		return true
	case strings.Contains(msg, "wsasendto"):
		return true
	case strings.Contains(msg, "timeout"):
		return true
	case strings.Contains(msg, "refused"):
		return true
	default:
		return false
	}
}

func (p *Peer) WriteContext(ctx context.Context, meta domain.AnyEvent, raw io.Reader) error {
	rawStream, err := p.conn.OpenStream(ctx)
	if err != nil {
		return fmt.Errorf("open stream: %w", err)
	}

	stream := &deadlineStream{
		stream: rawStream,
		read:   p.deadline.Read,
		write:  p.deadline.Write,
	}
	defer stream.Close()

	if err := protocol.WriteEvent(stream, meta); err != nil {
		return fmt.Errorf("write event: %w", err)
	}

	if raw != nil {
		if _, err := io.Copy(stream, raw); err != nil {
			return fmt.Errorf("write raw: %w", err)
		}
	}

	return nil
}

func (p *Peer) handleStream(ctx context.Context, rawStream transport.Stream) error {
	stream := &deadlineStream{
		stream: rawStream,
		read:   p.deadline.Read,
		write:  p.deadline.Write,
	}
	defer stream.Close()

	event, err := protocol.DecodeEvent(stream)
	if err != nil {
		return fmt.Errorf("decode event: %w", err)
	}

	switch payload := event.(type) {
	case domain.EventMessage:
		return p.handleMessage(payload, stream)
	case domain.EventAnnounce:
		p.channel.Announce(payload)
		return nil

	case domain.EventRequest:
		msg, ok := p.channel.Get(payload.Payload.ID)
		if !ok {
			p.logger.Debug().
				Int64("req_id", payload.Payload.ID.Int64()).
				Msg("peer requested message that i do not have or is expired")
			return nil
		}
		return p.handleRequest(ctx, msg, payload)

	default:
		return fmt.Errorf("unknown payload type: %T", payload)
	}
}

func (p *Peer) sendNack(msg domain.Message) {
	if msg.BatchID != 0 && p.batches != nil {
		msg.Data = nil
		p.batches.Add(msg)
	}
}

func (p *Peer) handleMessage(msg domain.EventMessage, stream transport.Stream) error {
	if msg.Payload.ContentLength > p.maxReceiveSize {
		p.sendNack(msg.Payload)
		return fmt.Errorf(
			"message size exceeds limit: %d > %d",
			msg.Payload.ContentLength,
			p.maxReceiveSize,
		)
	}

	if msg.Payload.MimeType.IsPath() {
		filePath, err := p.fileWriter.Write(stream, msg.Payload)
		if errors.Is(err, store.ErrFileExists) {
			_ = stream.Reset()
		} else if err != nil {
			p.sendNack(msg.Payload)
			return err
		}

		msg.Payload.Data = []byte(filePath)
	} else {
		data := make([]byte, msg.Payload.ContentLength)

		if _, err := io.ReadFull(stream, data); err != nil {
			p.sendNack(msg.Payload)
			return fmt.Errorf("read raw data: %w", err)
		}

		msg.Payload.Data = data
	}

	p.logger.Trace().
		Object("msg", msg.Payload).
		Msg("received message")

	p.channel.Send(msg)

	return nil
}

func (p *Peer) RequestMessage(ctx context.Context, id domain.MessageID) error {
	return p.WriteContext(ctx, domain.NewRequest(id), nil)
}

func (p *Peer) handleRequest(ctx context.Context, ev domain.EventMessage, req domain.EventRequest) error {
	ctxLog := ctxlog.Op(p.logger, "peer.handleRequest").With().Object("msg", ev.Payload).Logger()
	ctxLog.Trace().Msg("received request")

	if ev.Payload.ID != req.Payload.ID {
		return nil
	}

	ctxLog.Trace().Msg("sending")

	var r io.Reader

	if ev.Payload.MimeType.IsPath() {
		fp := string(ev.Payload.Data)
		file, err := os.Open(fp)
		if err != nil {
			return fmt.Errorf("failed to open file for streaming %s: %w", fp, err)
		}
		defer file.Close()
		r = file
	} else {
		r = bytes.NewReader(ev.Payload.Data)
	}

	err := p.WriteContext(ctx, ev, r)
	if errors.Is(err, transport.ErrStreamCanceled) {
		ctxLog.Trace().Msg("peer canceled receiving file")
		return nil
	}

	return err
}

type deadlineStream struct {
	stream    transport.Stream
	read      time.Duration
	write     time.Duration
	lastRead  time.Time
	lastWrite time.Time
}

func (s *deadlineStream) Read(p []byte) (int, error) {
	if s.read > 0 {
		now := time.Now()
		if now.Sub(s.lastRead) > s.read/4 {
			_ = s.stream.SetReadDeadline(now.Add(s.read))
			s.lastRead = now
		}
	}
	return s.stream.Read(p)
}

func (s *deadlineStream) Write(p []byte) (int, error) {
	if s.write > 0 {
		now := time.Now()
		if now.Sub(s.lastWrite) > s.write/4 {
			_ = s.stream.SetWriteDeadline(now.Add(s.write))
			s.lastWrite = now
		}
	}
	return s.stream.Write(p)
}

func (s *deadlineStream) Close() error {
	return s.stream.Close()
}

func (s *deadlineStream) SetReadDeadline(t time.Time) error {
	return s.stream.SetReadDeadline(t)
}

func (s *deadlineStream) SetWriteDeadline(t time.Time) error {
	return s.stream.SetWriteDeadline(t)
}

func (s *deadlineStream) Reset() error {
	return s.stream.Reset()
}
