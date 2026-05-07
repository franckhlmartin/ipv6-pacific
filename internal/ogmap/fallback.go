package ogmap

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

// FallbackPNG returns a minimal 1200×630 PNG when dynamic map generation fails.
func FallbackPNG() ([]byte, error) {
	w, h := ogWidth, ogHeight
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	c := color.RGBA{R: 30, G: 58, B: 138, A: 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			rgba.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, rgba); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
