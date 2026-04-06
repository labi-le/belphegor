package rfc8089

import "bytes"

const fileSchemePrefix = "file://"

// FormatURIList converts a newline-separated list of paths into a valid text/uri-list (RFC 2483)
// https://datatracker.ietf.org/doc/html/rfc8089
func FormatURIList(src []byte) []byte {
	if len(src) == 0 {
		return nil
	}

	lines := bytes.Split(src, []byte{'\n'})
	var formatted [][]byte

	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		if !bytes.HasPrefix(line, []byte(fileSchemePrefix)) {
			var b bytes.Buffer
			b.WriteString(fileSchemePrefix)
			b.Write(line)
			formatted = append(formatted, b.Bytes())
		} else {
			formatted = append(formatted, line)
		}
	}

	return bytes.Join(formatted, []byte("\r\n"))
}
