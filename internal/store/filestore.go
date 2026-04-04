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
		// Only treat as a cache hit when we have a non-zero hash to compare.
		// ContentHash == 0 is the unset sentinel in domain.Message; matching it
		// against a missing/corrupt sidecar (which also returns 0) would
		// incorrectly skip a needed download.
		if msg.ContentHash != 0 {
			if storedHash, ok := readHashFile(fullPath); ok &&
				uint64(info.Size()) == msg.ContentLength &&
				storedHash == msg.ContentHash {
				fs.logger.Trace().Str("path", fullPath).Msg("file already exists, skipping download")
				return fullPath, ErrFileExists
			}
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

	// Persist the hash sidecar so future cache lookups can detect updated
	// files that share the same name and size. On failure we keep the
	// downloaded file — it is valid — and log a warning. The missing sidecar
	// means readHashFile returns (0, false), the msg.ContentHash != 0 guard
	// prevents a false cache hit, and the file will be re-downloaded on the
	// next receive to get a fresh sidecar. Removing the file on sidecar
	// failure would create a TOCTOU race with concurrent ErrFileExists cache
	// hits that may already hold the path.
	if err := writeHashFile(fullPath, msg.ContentHash); err != nil {
		fs.logger.Warn().Err(err).Str("path", fullPath).
			Msg("filestore: failed to write hash sidecar; file kept, will re-verify on next receive")
	}
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

// readHashFile reads back the stored hash.
// Returns (hash, true) on success, (0, false) if the sidecar is absent or corrupt.
func readHashFile(filePath string) (uint64, bool) {
	b, err := os.ReadFile(hashFilePath(filePath))
	if err != nil || len(b) != 8 {
		return 0, false
	}
	return binary.LittleEndian.Uint64(b), true
}
