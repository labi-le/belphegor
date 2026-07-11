package mime_test

import (
	"testing"

	"github.com/labi-le/belphegor/pkg/mime"
)

func TestFrom_ByteSniff(t *testing.T) {
	filler := make([]byte, 32)
	tests := []struct {
		name string
		data []byte
		want mime.Type
	}{
		{"png magic", append([]byte{0x89, 0x50, 0x4E, 0x47}, filler...), mime.TypeImage},
		{"jpeg magic", append([]byte{0xFF, 0xD8}, filler...), mime.TypeImage},
		{"gif magic", append([]byte{0x47, 0x49, 0x46, 0x38}, filler...), mime.TypeImage},
		{"bmp magic", append([]byte{0x42, 0x4D}, filler...), mime.TypeImage},
		{"webp riff", append([]byte("RIFF\x00\x00\x00\x00WEBP"), filler...), mime.TypeImage},
		{"pdf classified binary", append([]byte{0x25, 0x50, 0x44, 0x46}, filler...), mime.TypeBinary},
		{"zip classified binary", append([]byte{0x50, 0x4B, 0x03, 0x04}, filler...), mime.TypeBinary},
		{"rar classified binary", append([]byte{0x52, 0x61, 0x72, 0x21}, filler...), mime.TypeBinary},
		{"gzip classified binary", append([]byte{0x1F, 0x8B}, filler...), mime.TypeBinary},
		{"plain text falls through", []byte("hello world"), mime.TypeText},
		{"empty is text", nil, mime.TypeText},
		{"too short for any magic is text", []byte{0x42}, mime.TypeText},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mime.From(tt.data); got != tt.want {
				t.Fatalf("From(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestAsType(t *testing.T) {
	tests := []struct {
		in   string
		want mime.Type
	}{
		{"text/plain", mime.TypeText},
		{"TEXT/PLAIN;charset=utf-8", mime.TypeText},
		{"image/png", mime.TypeImage},
		{"image/JPEG", mime.TypeImage},
		{"text/uri-list", mime.TypePath},
		{"application/x-cf-hdrop", mime.TypePath},
		{"application/octet-stream", mime.TypeUnknown},
		{"", mime.TypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := mime.AsType(tt.in); got != tt.want {
				t.Fatalf("AsType(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestIsSupported(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"text/plain", true},
		{"image/png", true},
		{"text/uri-list", true},
		{"IMAGE/PNG", true},
		{"application/octet-stream", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := mime.IsSupported(tt.in); got != tt.want {
				t.Fatalf("IsSupported(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestType_String(t *testing.T) {
	tests := []struct {
		typ  mime.Type
		want string
	}{
		{mime.TypeText, "text"},
		{mime.TypeImage, "image"},
		{mime.TypePath, "path"},
		{mime.TypeAudio, "audio"},
		{mime.TypeVideo, "video"},
		{mime.TypeBinary, "binary"},
		{mime.TypeUnknown, "unknown"},
		{mime.Type(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.typ.String(); got != tt.want {
			t.Errorf("Type(%d).String() = %q, want %q", tt.typ, got, tt.want)
		}
	}
}

func TestType_Predicates(t *testing.T) {
	if !mime.TypeImage.IsImage() || mime.TypeText.IsImage() {
		t.Error("IsImage predicate wrong")
	}
	if !mime.TypeText.IsText() || mime.TypeImage.IsText() {
		t.Error("IsText predicate wrong")
	}
	if !mime.TypePath.IsPath() || mime.TypeText.IsPath() {
		t.Error("IsPath predicate wrong")
	}
}

func TestSupportedTypes(t *testing.T) {
	got := mime.SupportedTypes()
	// 6 image + 5 text + 4 path labels registered in init.
	if len(got) != 15 {
		t.Errorf("SupportedTypes() has %d entries, want 15", len(got))
	}
	if got["image/png"] != mime.TypeImage {
		t.Errorf("SupportedTypes()[image/png] = %v, want image", got["image/png"])
	}
}
