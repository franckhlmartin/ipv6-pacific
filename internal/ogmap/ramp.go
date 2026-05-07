package ogmap

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// gradientHex matches cmd/web/static/js/pct-color-ramp.js GRADIENT — single definition order and hex strings.
var gradientHex = []struct {
	pct float64
	hex string
}{
	{0, "#FF0000"},
	{25, "#E04A24"},
	{50, "#8FA822"},
	{75, "#47C41B"},
	{100, "#00FF00"},
}

// NoDataGray matches pct-color-ramp.js NO_DATA_GRAY.
const NoDataGray = "#9aa3b2"

func hexToRGB(hex string) (r, g, b int) {
	h := strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(h) != 6 {
		return 0, 0, 0
	}
	r64, _ := strconv.ParseUint(h[0:2], 16, 8)
	g64, _ := strconv.ParseUint(h[2:4], 16, 8)
	b64, _ := strconv.ParseUint(h[4:6], 16, 8)
	return int(r64), int(g64), int(b64)
}

// ColorForPct mirrors pct-color-ramp.js colorForPct (same stops, interpolation, Math.round).
// Returns 6-digit lowercase hex for SVG (browser often uses rgb(...) mid-ramp; numerically identical).
func ColorForPct(p float64) string {
	p = math.Max(0, math.Min(100, p))
	stops := gradientHex
	if p <= stops[0].pct {
		return strings.ToLower(stops[0].hex)
	}
	if p >= stops[len(stops)-1].pct {
		return strings.ToLower(stops[len(stops)-1].hex)
	}
	i := 0
	for i < len(stops)-1 && p > stops[i+1].pct {
		i++
	}
	a, b := stops[i], stops[i+1]
	u := (p - a.pct) / (b.pct - a.pct)
	caR, caG, caB := hexToRGB(a.hex)
	cbR, cbG, cbB := hexToRGB(b.hex)
	r := int(math.Round(float64(caR) + float64(cbR-caR)*u))
	g := int(math.Round(float64(caG) + float64(cbG-caG)*u))
	bl := int(math.Round(float64(caB) + float64(cbB-caB)*u))
	return fmt.Sprintf("#%02x%02x%02x", r, g, bl)
}
