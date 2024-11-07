package domain

import (
	"github.com/google/uuid"
	"time"
)

type Header struct {
	ID                uuid.UUID
	Created           time.Time
	From              UniqueID
	MimeType          MimeType
	ClipboardProvider ClipboardProvider
}

func NewHeader(opts ...Option) Header {
	header := &Header{
		ID:                uuid.New(),
		Created:           time.Now(),
		From:              SelfMetaData().UniqueID(),
		ClipboardProvider: CurrentClipboardProvider,
	}
	for _, opt := range opts {
		opt(header)
	}

	return *header
}

type Option func(header *Header)

func WithClipboardProvider(cp ClipboardProvider) Option {
	return func(header *Header) {
		header.ClipboardProvider = cp
	}
}

func WithMime(mime MimeType) Option {
	return func(header *Header) {
		header.MimeType = mime
	}
}
