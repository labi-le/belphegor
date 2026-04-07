package eventful

import (
	"context"
	"fmt"

	"github.com/dustin/go-humanize"
	"github.com/labi-le/belphegor/pkg/mime"
	"github.com/rs/zerolog"
)

type Eventful interface {
	// Watch subscribe to clipboard updates
	// when the context is finished, Watch must close the update channel
	Watch(ctx context.Context, upd chan<- Update) error
	Write(t mime.Type, src []byte) (int, error)
}

type Update struct {
	// Data or path to data. if is a path, then, according to the contract,
	// it is necessary to return clean path without a trash
	Data       []byte
	Size       uint64
	MimeType   mime.Type
	Hash       uint64
	BatchID    uint64
	BatchTotal uint32
}

func (u Update) MarshalZerologObject(e *zerolog.Event) {
	e.Uint64("length", u.Size)
	e.Uint64("hash", u.Hash)
	e.Stringer("mime", u.MimeType)
	e.Uint64("batch_id", u.BatchID)
	e.Uint32("batch_total", u.BatchTotal)
}

type Options struct {
	AllowCopyFiles    bool
	MaxFileSize       MaxFileSize
	MaxClipboardFiles int
}

type MaxFileSize uint64

func (m MaxFileSize) String() string {
	return humanize.Bytes(uint64(m))
}

func (m *MaxFileSize) Set(s string) error {
	size, err := humanize.ParseBytes(s)
	if err != nil {
		return fmt.Errorf("invalid max_file_size: %w", err)
	}
	*m = MaxFileSize(size)
	return nil
}

func (m *MaxFileSize) Type() string {
	return "uint64"
}
