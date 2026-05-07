package ogmap

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

const (
	ogWidth  = 1200
	ogHeight = 630
)

// RasterizeEEZToPNG renders modified EEZ SVG to PNG (1200×630, neutral letterbox).
func RasterizeEEZToPNG(svgXML []byte) ([]byte, error) {
	svgXML = SanitizeSVGColorsOKSVG(svgXML)
	icon, err := oksvg.ReadIconStream(bytes.NewReader(svgXML), oksvg.IgnoreErrorMode)
	if err != nil {
		return nil, fmt.Errorf("oksvg: %w", err)
	}
	w, h := ogWidth, ogHeight
	icon.SetTarget(0, 0, float64(w), float64(h))
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	bg := color.RGBA{R: 243, G: 244, B: 246, A: 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			rgba.Set(x, y, bg)
		}
	}
	sc := rasterx.NewScannerGV(w, h, rgba, rgba.Bounds())
	rz := rasterx.NewDasher(w, h, sc)
	icon.Draw(rz, 1.0)
	var buf bytes.Buffer
	if err := png.Encode(&buf, rgba); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
