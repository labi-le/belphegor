package image

import (
	"image"
	"image/png"
	"io"
	"math"
)

const threshold = 1000.0

func calculateMSE(img1 image.Image, img2 image.Image) float64 {
	if img1.Bounds() != img2.Bounds() {
		return math.MaxFloat64
	}

	bounds := img1.Bounds()
	mse := 0.0

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			p1 := img1.At(x, y)
			p2 := img2.At(x, y)
			r1, g1, b1, a1 := p1.RGBA()
			r2, g2, b2, a2 := p2.RGBA()
			dr := float64(r1) - float64(r2)
			dg := float64(g1) - float64(g2)
			db := float64(b1) - float64(b2)
			da := float64(a1) - float64(a2)
			mse += dr*dr + dg*dg + db*db + da*da
		}
	}

	mse /= float64(bounds.Dx() * bounds.Dy())
	return mse
}

// EqualMSE comparing two images by mean square error (MSE)
func EqualMSE(img1 io.Reader, img2 io.Reader) (bool, error) {
	png1, err := png.Decode(img1)
	if err != nil {
		return false, err
	}

	png2, err := png.Decode(img2)
	if err != nil {
		return false, err
	}
	// calculate mse between images
	return calculateMSE(png1, png2) < threshold, nil
}
