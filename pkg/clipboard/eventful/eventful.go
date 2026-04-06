package eventful

import (
	"context"

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
	MaxFileSize       uint64
	MaxClipboardFiles int
}
