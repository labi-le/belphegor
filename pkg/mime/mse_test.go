package mime_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/labi-le/belphegor/pkg/mime"
)

func pngBytes(t *testing.T, w, h int, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestEqualMSE(t *testing.T) {
	black := pngBytes(t, 8, 8, color.RGBA{R: 0, G: 0, B: 0, A: 255})
	blackCopy := pngBytes(t, 8, 8, color.RGBA{R: 0, G: 0, B: 0, A: 255})
	white := pngBytes(t, 8, 8, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	biggerBlack := pngBytes(t, 16, 16, color.RGBA{R: 0, G: 0, B: 0, A: 255})

	t.Run("identical images are equal", func(t *testing.T) {
		eq, err := mime.EqualMSE(bytes.NewReader(black), bytes.NewReader(blackCopy))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !eq {
			t.Fatal("identical images must compare equal")
		}
	})

	t.Run("very different images are not equal", func(t *testing.T) {
		eq, err := mime.EqualMSE(bytes.NewReader(black), bytes.NewReader(white))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if eq {
			t.Fatal("black vs white must exceed MSE threshold")
		}
	})

	t.Run("different dimensions are not equal", func(t *testing.T) {
		eq, err := mime.EqualMSE(bytes.NewReader(black), bytes.NewReader(biggerBlack))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if eq {
			t.Fatal("mismatched bounds must compare not equal")
		}
	})

	t.Run("first reader not a png errors", func(t *testing.T) {
		if _, err := mime.EqualMSE(bytes.NewReader([]byte("not a png")), bytes.NewReader(black)); err == nil {
			t.Fatal("expected decode error for invalid first image")
		}
	})

	t.Run("second reader not a png errors", func(t *testing.T) {
		if _, err := mime.EqualMSE(bytes.NewReader(black), bytes.NewReader([]byte("not a png"))); err == nil {
			t.Fatal("expected decode error for invalid second image")
		}
	})
}
