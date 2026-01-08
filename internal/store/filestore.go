package store

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/pool/byteslice"
	"github.com/rs/zerolog"
)

const bufSize = 1024 << 10

type FileStore struct {
	baseDir string
	logger  zerolog.Logger
}

func NewFileStore(baseDir string, logger zerolog.Logger) *FileStore {
	with := logger.With().Str("component", "filestore").Logger()
	with.Trace().Str("baseDir", baseDir).Msg("file save path")

	return &FileStore{
		baseDir: baseDir,
		logger:  with,
	}
}

func (fs *FileStore) Write(r io.Reader, msg domain.Message) (string, error) {
	fullPath := filepath.Join(fs.baseDir, msg.Name)

	buf := byteslice.Get(bufSize)
	defer byteslice.Put(buf)

	if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
		if uint64(info.Size()) == msg.ContentLength {
			fs.logger.Trace().Str("path", fullPath).Msg("file already exists, skipping download")
			_, _ = io.CopyBuffer(io.Discard, io.LimitReader(r, int64(msg.ContentLength)), buf)
			return fullPath, nil
		}
	}

	if err := os.MkdirAll(fs.baseDir, 0755); err != nil {
		return "", fmt.Errorf("filestore mkdir: %w", err)
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("filestore create file: %w", err)
	}
	defer file.Close()

	n, err := io.CopyBuffer(file, io.LimitReader(r, int64(msg.ContentLength)), buf)
	if err != nil {
		_ = os.Remove(fullPath)
		return "", fmt.Errorf("filestore write content: %w", err)
	}

	if uint64(n) != msg.ContentLength {
		_ = os.Remove(fullPath)
		return "", fmt.Errorf("filestore incomplete write: expected %d, got %d", msg.ContentLength, n)
	}

	fs.logger.Trace().Str("path", fullPath).Msg("file saved")
	return fullPath, nil
}
