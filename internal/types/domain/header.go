package domain

import (
	"time"
)

type Header struct {
	ID                UniqueID
	Created           time.Time
	From              UniqueID
	MimeType          MimeType
	ClipboardProvider ClipboardProvider
}

func NewHeader(from UniqueID, mime MimeType) Header {
	return Header{
		ID:                NewID(),
		Created:           time.Now(),
		From:              from,
		MimeType:          mime,
		ClipboardProvider: CurrentClipboardProvider,
	}
}
