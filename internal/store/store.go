package store

import (
	"io"

	"github.com/labi-le/belphegor/internal/types/domain"
)

type FileWriter interface {
	Write(r io.Reader, msg domain.Message) (string, error)
}
