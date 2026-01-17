package eventful

import (
	"bytes"
	"encoding/binary"
	"net/url"
	"os"
	"unsafe"

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

		_, _ = batchDigest.Write(buf)

		updates = append(updates, Update{
			Data:     unsafe.Slice(unsafe.StringData(file.Path), len(file.Path)),
			Size:     file.Size,
			MimeType: mime.TypePath,
			Hash:     xxhash.Sum64(buf),
		})

		buf = buf[:0]
	}

	return updates, batchDigest.Sum(nil)
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

		idx := bytes.IndexByte(data, '\n')
		var line []byte
		if idx >= 0 {
			line = data[:idx]
			data = data[idx+1:]
		} else {
			line = data
			data = nil
		}

		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}

		if len(line) == 0 || !bytes.HasPrefix(line, []byte("file://")) {
			continue
		}

		pathBytes := line[7:]

		path := strutil.BytesToString(pathBytes)

		if bytes.IndexByte(pathBytes, '%') >= 0 {
			if unescaped, err := url.PathUnescape(path); err == nil {
				path = unescaped
			}
		}

		info, err := os.Lstat(path)
		if err != nil {
			continue
		}

		if info.IsDir() {
			continue
		}

		res = append(res, FileInfo{
			Path:    path,
			Size:    uint64(info.Size()),
			ModTime: uint64(info.ModTime().UnixNano()),
		})
	}

	return res
}

func UpdatesFromRawPath(data []byte, limit int) ([]Update, []byte) {
	raw := fileInfoFromRaw(data, limit)
	if len(raw) == 0 {
		return []Update{}, nil
	}

	return UpdatesFromFileInfo(raw)
}
