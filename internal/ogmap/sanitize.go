package ogmap

import "regexp"

var (
	reHexLen8 = regexp.MustCompile(`#[0-9a-fA-F]{8}`)
	reHexLen7 = regexp.MustCompile(`#[0-9a-fA-F]{7}`)
)

// SanitizeSVGColorsOKSVG normalizes #RRGGBBAA and malformed 7-digit hex so github.com/srwiley/oksvg
// (3- or 6-digit #hex only) can parse fills and strokes. Apply 8-digit patterns before 7-digit.
func SanitizeSVGColorsOKSVG(svgXML []byte) []byte {
	s := string(svgXML)
	s = reHexLen8.ReplaceAllStringFunc(s, func(m string) string {
		if len(m) != 9 { // # + 8 hex
			return m
		}
		return m[:7] // # + 6 hex (drop alpha / tail)
	})
	s = reHexLen7.ReplaceAllStringFunc(s, func(m string) string {
		if len(m) != 8 { // # + 7 hex
			return m
		}
		return m[:7]
	})
	return []byte(s)
}
