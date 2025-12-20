package domain_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/labi-le/belphegor/internal/types/domain"
)

func TestMessage_Duplicate(t *testing.T) {

	img1 := createTestImage(t, 50, 50, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	img2 := createTestImage(t, 50, 50, color.RGBA{R: 0, G: 255, B: 0, A: 255})

	tests := []struct {
		name string
		msg  domain.EventMessage
		new  domain.EventMessage
		want bool
	}{
		{
			name: "same message reference",
			msg:  domain.MessageFrom([]byte(", 1test"), 1),
			new:  domain.MessageFrom([]byte(", 1test"), 1),
			want: true,
		},
		{
			name: "same text content",
			msg:  domain.MessageFrom([]byte(", 1test"), 1),
			new:  domain.MessageFrom([]byte(", 1test"), 2),
			want: true,
		},
		{
			name: "different text content",
			msg:  domain.MessageFrom([]byte(", 1test1"), 1),
			new:  domain.MessageFrom([]byte(", 1test2"), 2),
			want: false,
		},
		{
			name: "same image different source",
			msg:  domain.MessageFrom(img1, 1),
			new:  domain.MessageFrom(img1, 2),
			want: true,
		},
		{
			name: "different images",
			msg:  domain.MessageFrom(img1, 1),
			new:  domain.MessageFrom(img2, 1),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "same message reference" {
				tt.new = tt.msg
			}

			if got := tt.msg.Payload.Duplicate(tt.new.Payload); got != tt.want {
				t.Errorf("%s: Message.Duplicate() = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func createTestImage(t *testing.T, width, height int, c color.Color) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, c)
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal("failed to encode png:", err)
	}

	return buf.Bytes()
}

func BenchmarkMessage_Duplicate(b *testing.B) {
	type raw struct {
		data []byte
		id   domain.UniqueID
	}
	benchmarks := []struct {
		name     string
		msg, new raw
	}{
		{
			name: "small_text_same",
			msg:  raw{data: []byte("test")},
			new:  raw{data: []byte("test")},
		},
		{
			name: "small_text_different",
			msg:  raw{data: []byte("test")},
			new:  raw{data: []byte("different")},
		},
		{
			name: "large_text_different",
			msg:  raw{data: []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.")},
			new:  raw{data: []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat")},
		},
		{
			name: "small_image_different",
			msg:  raw{data: makeImageMsg(100, 100, color.RGBA{R: 255})},
			new:  raw{data: makeImageMsg(100, 100, color.RGBA{G: 255})},
		},
		{
			name: "medium_image_different",
			msg:  raw{data: makeImageMsg(800, 600, color.RGBA{R: 255})},
			new:  raw{data: makeImageMsg(800, 600, color.RGBA{G: 255})},
		},
		{
			name: "large_image_different",
			msg:  raw{data: makeImageMsg(1920, 1080, color.RGBA{R: 255})},
			new:  raw{data: makeImageMsg(1920, 1080, color.RGBA{G: 255})},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				msg := domain.MessageFrom(bm.msg.data, 1)
				msg.Payload.Duplicate(domain.MessageFrom(bm.new.data, 1).Payload)
			}
		})
	}
}

func makeImageMsg(width, height int, c color.Color) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, c)
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)

	return buf.Bytes()
}
