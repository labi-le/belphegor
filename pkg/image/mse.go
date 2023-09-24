package image

import (
	"bytes"
	"image"
	"image/png"
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

// IsDuplicate comparing two images by mean square error (MSE)
func IsDuplicate(imageData1 []byte, imageData2 []byte) (bool, error) {
	img1, err := png.Decode(bytes.NewReader(imageData1))
	if err != nil {
		return false, err
	}

	img2, err := png.Decode(bytes.NewReader(imageData2))
	if err != nil {
		return false, err
	}
	// calculate mse between images
	mse := calculateMSE(img1, img2)

	return mse < threshold, nil
}
