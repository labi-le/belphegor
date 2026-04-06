package store

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/pool/byteslice"
	"github.com/rs/zerolog"
)

const bufSize = 1024 << 10

type FileStore struct {
	baseDir string
	logger  zerolog.Logger
}

func NewFileStore(baseDir string, logger zerolog.Logger) (*FileStore, error) {
	with := logger.With().Str("component", "filestore").Logger()
	with.Trace().Str("baseDir", baseDir).Msg("file save path")

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("filestore mkdir: %w", err)
	}

	return &FileStore{
		baseDir: baseDir,
		logger:  with,
	}, nil
}

func MustFileStore(baseDir string, logger zerolog.Logger) *FileStore {
	store, err := NewFileStore(baseDir, logger)
	if err != nil {
		panic(err)
	}

	return store
}

func (fs *FileStore) Write(r io.Reader, msg domain.Message) (string, error) {
	if msg.Name == "" {
		return "", fmt.Errorf("invalid filename: name is empty")
	}

	var isolateDir string
	if msg.BatchID != 0 {
		isolateDir = filepath.Join(fs.baseDir, msg.BatchID.String())
	} else {
		isolateDir = filepath.Join(fs.baseDir, msg.ID.String())
	}

	isolateDirClean := filepath.Clean(isolateDir)
	if err := os.MkdirAll(isolateDirClean, 0755); err != nil {
		return "", fmt.Errorf("filestore mkdir isolated: %w", err)
	}

	fullPath := filepath.Join(isolateDirClean, msg.Name)

	if !strings.HasPrefix(fullPath, isolateDirClean+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid filename: path traversal attempt detected (%q)", msg.Name)
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("filestore mkdir tree: %w", err)
	}

	buf := byteslice.Get(bufSize)
	defer byteslice.Put(buf)

	if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
		if uint64(info.Size()) == msg.ContentLength {
			fs.logger.Trace().Str("path", fullPath).Msg("file already exists, skipping download")
			return fullPath, ErrFileExists
		}
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
