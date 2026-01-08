package store

import (
	"errors"
	"io"

	"github.com/labi-le/belphegor/internal/types/domain"
)

var ErrFileExists = errors.New("file already exists")

type FileWriter interface {
	Write(r io.Reader, msg domain.Message) (string, error)
}
