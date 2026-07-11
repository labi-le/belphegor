package rfc8089_test

import (
	"testing"

	"github.com/labi-le/belphegor/pkg/rfc8089"
)

func TestFormatURIList(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"single bare path gets file scheme", "/home/user/a.txt", "file:///home/user/a.txt"},
		{"already prefixed path kept as-is", "file:///home/user/a.txt", "file:///home/user/a.txt"},
		{"two bare paths joined with CRLF", "/a\n/b", "file:///a\r\nfile:///b"},
		{"blank lines skipped, trailing newline ignored", "/a\n\n/b\n", "file:///a\r\nfile:///b"},
		{"surrounding whitespace trimmed", "  /a  \n\t/b\t", "file:///a\r\nfile:///b"},
		{"mixed prefixed and bare", "file:///a\n/b", "file:///a\r\nfile:///b"},
		{"whitespace-only yields empty", "  \n\t\n", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(rfc8089.FormatURIList([]byte(tt.in)))
			if got != tt.want {
				t.Fatalf("FormatURIList(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestFormatURIList_EmptyInputReturnsNil(t *testing.T) {
	if got := rfc8089.FormatURIList(nil); got != nil {
		t.Errorf("FormatURIList(nil) = %v, want nil", got)
	}
	if got := rfc8089.FormatURIList([]byte{}); got != nil {
		t.Errorf("FormatURIList([]byte{}) = %v, want nil", got)
	}
}
