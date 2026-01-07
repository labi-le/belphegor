package peer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/labi-le/belphegor/internal/channel"
	"github.com/labi-le/belphegor/internal/protocol"
	"github.com/labi-le/belphegor/internal/transport"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/ctxlog"
	"github.com/labi-le/belphegor/pkg/id"
	"github.com/labi-le/belphegor/pkg/network"
	"github.com/rs/zerolog"
)

type Peer struct {
	conn       transport.Connection
	metaData   domain.Device
	channel    *channel.Channel
	stringRepr string
	logger     zerolog.Logger
	deadline   network.Deadline
}

func New(
	conn transport.Connection,
	metadata domain.Device,
	channel *channel.Channel,
	logger zerolog.Logger,
	dd network.Deadline,
) *Peer {
	return &Peer{
		conn:       conn,
		metaData:   metadata,
		channel:    channel,
		logger:     logger,
		deadline:   dd,
		stringRepr: fmt.Sprintf("%s -> %s", metadata.Name, conn.RemoteAddr().String()),
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

func (p *Peer) Receive(ctx context.Context, savePath string) error {
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
				ctxLog.Error().Err(err).Msg("failed to accept stream")
				continue
			}

			if handleErr := p.handleStream(ctx, stream, savePath); handleErr != nil {
				ctxLog.Trace().Err(handleErr).Msg("failed to handle stream")
			}
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

	msg := err.Error()
	return strings.Contains(msg, "closed") || strings.Contains(msg, "application error 0x0")
}

func (p *Peer) WriteContext(ctx context.Context, meta any, raw []byte) error {
	stream, err := p.conn.OpenStream(ctx)
	if err != nil {
		return fmt.Errorf("open stream: %w", err)
	}

	defer func(stream transport.Stream) {
		if err := stream.Close(); err != nil {
			p.logger.Trace().Err(err).Msg("failed to close writer stream")
		}
	}(stream)

	if err := network.SetWriteDeadline(stream, p.deadline); err != nil {
		return err
	}

	if err := protocol.WriteEvent(stream, meta); err != nil {
		return fmt.Errorf("write event: %w", err)
	}

	if len(raw) > 0 {
		if _, err := stream.Write(raw); err != nil {
			return fmt.Errorf("write raw: %w", err)
		}
	}

	return nil
}

func (p *Peer) handleStream(ctx context.Context, stream transport.Stream, path string) error {
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
		return p.handleMessage(payload, stream, path)

	case domain.EventAnnounce:
		p.channel.Announce(payload)
		p.logger.Trace().
			Int64("msg_id", payload.Payload.ID).
			Msg("received announce")
		return nil

	case domain.EventRequest:
		return p.handleRequest(ctx, p.channel.LastMsg(), payload)

	default:
		return fmt.Errorf("unknown payload type: %T", payload)
	}
}

func (p *Peer) handleMessage(msg domain.EventMessage, reader io.Reader, path string) error {
	if msg.Payload.MimeType.IsPath() {
		filePath, err := p.createOrGetCachedFile(path, msg.Payload, reader)
		if err != nil {
			p.logger.Error().Err(err).Msg("failed to handle incoming file")
			return err
		}
		msg.Payload.Data = []byte(filePath)
	} else {
		data := make([]byte, msg.Payload.ContentLength)

		if _, err := io.ReadFull(reader, data); err != nil {
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

	if ev.Payload.MimeType.IsPath() {
		return p.streamFile(ctx, ev)
	}

	meta := ev
	// data written separately to stream
	meta.Payload.Data = nil

	return p.WriteContext(ctx, meta, ev.Payload.Data)
}

func (p *Peer) streamFile(ctx context.Context, meta domain.EventMessage) error {
	fp := string(meta.Payload.Data)
	file, err := os.Open(fp)
	if err != nil {
		return fmt.Errorf("failed to open file for streaming %s: %w", fp, err)
	}
	defer file.Close()

	stream, err := p.conn.OpenStream(ctx)
	if err != nil {
		return fmt.Errorf("open stream for file: %w", err)
	}
	defer func(stream transport.Stream) {
		if err := stream.Close(); err != nil {
			p.logger.Trace().Err(err).Msg("failed to close writer stream")
		}
	}(stream)

	if err := network.SetWriteDeadline(stream, p.deadline); err != nil {
		return err
	}

	if err := protocol.WriteEvent(stream, meta); err != nil {
		return fmt.Errorf("write file event: %w", err)
	}

	bytesSent, err := io.Copy(stream, file)
	if err != nil {
		return fmt.Errorf("write file raw data: %w", err)
	}

	p.logger.Trace().
		Int64("bytes_sent", bytesSent).
		Msg("file stream sent successfully")

	return nil
}

func (p *Peer) createOrGetCachedFile(path string, msg domain.Message, reader io.Reader) (string, error) {
	filePath := filepath.Join(path, msg.Name)

	if _, err := os.Stat(filePath); err == nil {
		p.logger.Trace().
			Str("file_path", filePath).
			Msg("file already exists in cache, skipping download")

		_, _ = io.CopyN(io.Discard, reader, int64(msg.ContentLength))

		return filePath, nil
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create cache file: %w", err)
	}
	defer file.Close()

	_, err = io.CopyN(file, reader, int64(msg.ContentLength))
	if err != nil {
		_ = os.Remove(filePath)
		return "", fmt.Errorf("failed to write stream to cache file: %w", err)
	}

	p.logger.Trace().
		Str("file_path", filePath).
		Msg("received file and saved to cache")

	return filePath, nil
}
