package store

import (
	"encoding/binary"
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
	fullPath := filepath.Join(fs.baseDir, msg.Name)

	buf := byteslice.Get(bufSize)
	defer byteslice.Put(buf)

	if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
		if uint64(info.Size()) == msg.ContentLength && readHashFile(fullPath) == msg.ContentHash {
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

	_ = writeHashFile(fullPath, msg.ContentHash)
	fs.logger.Trace().Str("path", fullPath).Msg("file saved")
	return fullPath, nil
}

// hashFilePath returns the sidecar file path used to store the content hash.
func hashFilePath(filePath string) string {
	return filePath + ".bfghash"
}

// writeHashFile persists the 8-byte little-endian hash alongside the cached file.
func writeHashFile(filePath string, hash uint64) error {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], hash)
	return os.WriteFile(hashFilePath(filePath), b[:], 0600)
}

// readHashFile reads back the stored hash. Returns 0 if the sidecar is absent or corrupt.
func readHashFile(filePath string) uint64 {
	b, err := os.ReadFile(hashFilePath(filePath))
	if err != nil || len(b) != 8 {
		return 0
	}
	return binary.LittleEndian.Uint64(b)
}
