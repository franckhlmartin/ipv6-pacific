package ogmap

import (
	"crypto/sha256"
	"encoding/hex"
)

// TagVersion bumps when OG SVG mutation logic changes (invalidates ETags).
const TagVersion = "ogmap-v3"

// BuildMapPNG returns PNG bytes and a stable ETag string (hex, no quotes) from index + SVG inputs.
func BuildMapPNG(indexJSON, svgXML []byte) (png []byte, etag string, err error) {
	pref, err := PreferredFromIndexJSON(indexJSON)
	if err != nil {
		return nil, "", err
	}
	svg2, err := ApplyPreferredToSVG(svgXML, pref)
	if err != nil {
		return nil, "", err
	}
	sum := sha256.New()
	sum.Write([]byte(TagVersion))
	sum.Write(indexJSON)
	sum.Write(svg2)
	etag = hex.EncodeToString(sum.Sum(nil))
	png, err = RasterizeEEZToPNG(svg2)
	if err != nil {
		return nil, "", err
	}
	return png, etag, nil
}
