package store

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/rs/zerolog"
)

type FileStore struct {
	baseDir string
	logger  zerolog.Logger
}

func NewFileStore(baseDir string, logger zerolog.Logger) *FileStore {
	return &FileStore{
		baseDir: baseDir,
		logger:  logger,
	}
}

func (fs *FileStore) Write(r io.Reader, msg domain.Message) (string, error) {
	fullPath := filepath.Join(fs.baseDir, msg.Name)

	if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
		if uint64(info.Size()) == msg.ContentLength {
			fs.logger.Trace().Str("path", fullPath).Msg("file already exists, skipping download")
			_, _ = io.CopyN(io.Discard, r, int64(msg.ContentLength))
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

	_, err = io.CopyN(file, r, int64(msg.ContentLength))
	if err != nil {
		_ = os.Remove(fullPath)
		return "", fmt.Errorf("filestore write content: %w", err)
	}

	fs.logger.Trace().Str("path", fullPath).Msg("file saved")
	return fullPath, nil
}
