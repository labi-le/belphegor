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

	"github.com/labi-le/belphegor/internal/channel"
	"github.com/labi-le/belphegor/internal/protocol"
	"github.com/labi-le/belphegor/internal/store"
	"github.com/labi-le/belphegor/internal/transport"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/ctxlog"
	"github.com/labi-le/belphegor/pkg/id"
	"github.com/labi-le/belphegor/pkg/network"
	"github.com/rs/zerolog"
)

type Options struct {
	Channel        *channel.Channel
	Store          store.FileWriter
	Logger         zerolog.Logger
	Deadline       network.Deadline
	MaxReceiveSize uint64
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
	stream, err := p.conn.OpenStream(ctx)
	if err != nil {
		return fmt.Errorf("open stream: %w", err)
	}

	defer func(stream transport.Stream) { _ = stream.Close() }(stream)

	if err := network.SetWriteDeadline(stream, p.deadline); err != nil {
		return err
	}

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

func (p *Peer) handleStream(ctx context.Context, stream transport.Stream) error {
	defer stream.Close()

	if err := network.SetReadDeadline(stream, p.deadline); err != nil {
		return err
	}

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
		return p.handleRequest(ctx, p.channel.LastMsg(), payload)

	default:
		return fmt.Errorf("unknown payload type: %T", payload)
	}
}

func (p *Peer) handleMessage(msg domain.EventMessage, stream transport.Stream) error {
	if msg.Payload.ContentLength > p.maxReceiveSize {
		return fmt.Errorf(
			"message size exceeds limit: %d > %d",
			msg.Payload.ContentLength,
			p.maxReceiveSize,
		)
	}

	if msg.Payload.MimeType.IsPath() {
		filePath, err := p.fileWriter.Write(stream, msg.Payload)
		if err != nil {
			if errors.Is(err, store.ErrFileExists) && stream.Reset() == nil {
				return nil
			}

			return err
		}
		msg.Payload.Data = []byte(filePath)
	} else {
		data := make([]byte, msg.Payload.ContentLength)

		if _, err := io.ReadFull(stream, data); err != nil {
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

func (p *Peer) Request(ctx context.Context, messageID id.Unique) error {
	req := domain.NewRequest(messageID)
	p.logger.Trace().Int64("msg_id", messageID).Msg("sending request packet")

	return p.WriteContext(ctx, req, nil)
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
