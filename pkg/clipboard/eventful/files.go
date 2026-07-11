package eventful

import (
	"bytes"
	"encoding/binary"
	"net/url"
	"os"

	"github.com/cespare/xxhash"
	"github.com/labi-le/belphegor/pkg/mime"
	"github.com/labi-le/belphegor/pkg/strutil"
)

type FileInfo struct {
	Path    string
	Size    uint64
	ModTime uint64
}

func UpdatesFromFileInfo(files []FileInfo) ([]Update, []byte) {
	updates := make([]Update, 0, len(files))

	batchDigest := xxhash.New()

	buf := make([]byte, 0, 512)

	for _, file := range files {
		buf = append(buf, file.Path...)
		buf = binary.LittleEndian.AppendUint64(buf, file.Size)
		buf = binary.LittleEndian.AppendUint64(buf, file.ModTime)
	}

	_, _ = batchDigest.Write(buf)

	batchHashBytes := batchDigest.Sum(nil)
	batchID := batchDigest.Sum64() & 0x7FFFFFFFFFFFFFFF
	batchTotal := uint32(len(files))

	buf = buf[:0]

	for _, file := range files {
		buf = append(buf, file.Path...)
		buf = binary.LittleEndian.AppendUint64(buf, file.Size)
		buf = binary.LittleEndian.AppendUint64(buf, file.ModTime)

		updates = append(updates, Update{
			Data:       strutil.StringToBytes(file.Path),
			Size:       file.Size,
			MimeType:   mime.TypePath,
			Hash:       xxhash.Sum64(buf),
			BatchID:    batchID,
			BatchTotal: batchTotal,
		})

		buf = buf[:0]
	}

	return updates, batchHashBytes
}

func fileInfoFromRaw(data []byte, limit int) []FileInfo {
	capSize := 8
	if limit > 0 {
		capSize = limit
	}
	res := make([]FileInfo, 0, capSize)

	for len(data) > 0 {
		if limit > 0 && len(res) >= limit {
			break
		}

		var line []byte
		line, data = nextLine(data)

		if info, ok := parseFileURI(line); ok {
			res = append(res, info)
		}
	}

	return res
}

func nextLine(data []byte) (line, rest []byte) {
	line, rest, _ = bytes.Cut(data, []byte{'\n'})

	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}

	return line, rest
}

func parseFileURI(line []byte) (FileInfo, bool) {
	if len(line) == 0 || !bytes.HasPrefix(line, []byte("file://")) {
		return FileInfo{}, false
	}

	pathBytes := line[7:]
	path := strutil.BytesToString(pathBytes)
	if bytes.IndexByte(pathBytes, '%') >= 0 {
		if unescaped, err := url.PathUnescape(path); err == nil {
			path = unescaped
		}
	}

	info, err := os.Lstat(path)
	if err != nil || info.IsDir() {
		return FileInfo{}, false
	}

	return FileInfo{
		Path:    path,
		Size:    uint64(info.Size()),
		ModTime: uint64(info.ModTime().UnixNano()),
	}, true
}

func UpdatesFromRawPath(data []byte, limit int) ([]Update, []byte) {
	raw := fileInfoFromRaw(data, limit)
	if len(raw) == 0 {
		return []Update{}, nil
	}

	return UpdatesFromFileInfo(raw)
}
