package ogmap

import (
	"fmt"

	"github.com/beevik/etree"
)

const maxSVGBytes = 4 << 20 // 4 MiB cap (plan: bounded input)

// ApplyPreferredToSVG edits the embedded EEZ SVG like map-home.js: ocean rect, viewBox,
// path fills from percentage map, grey for titled non-monitored regions.
// Uses SVG presentation attributes (fill, stroke) — github.com/srwiley/oksvg applies these
// reliably; inline style alone was easy to lose during rasterization.
func ApplyPreferredToSVG(svgXML []byte, preferred map[string]float64) ([]byte, error) {
	if len(svgXML) > maxSVGBytes {
		return nil, fmt.Errorf("svg exceeds max size")
	}
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(svgXML); err != nil {
		return nil, err
	}
	root := doc.Root()
	if root == nil || root.Tag != "svg" {
		return nil, fmt.Errorf("root is not svg")
	}

	defs := firstChildTag(root, "defs")
	ocean := findByID(root, "rect5538-5")
	if ocean != nil && defs != nil {
		idx := indexOfChild(root, defs)
		if idx < 0 {
			idx = 0
		}
		ocean.CreateAttr("x", "0")
		ocean.CreateAttr("y", "0")
		ocean.CreateAttr("width", "385")
		ocean.CreateAttr("height", "215")
		ocean.RemoveAttr("style")
		ocean.CreateAttr("fill", "#c6ecff")
		root.InsertChildAt(idx+1, ocean)
	}

	root.CreateAttr("viewBox", "0 0 385 215")
	root.CreateAttr("preserveAspectRatio", "xMidYMid meet")
	root.RemoveAttr("width")
	root.RemoveAttr("height")

	for _, path := range collectTag(root, "path") {
		titleText := childTextTitle(path)
		if titleText == "" {
			continue
		}
		iso := ISOForTerritoryTitle(titleText)
		clearPathPaint(path)
		if iso == "" {
			path.CreateAttr("fill", "#b8bcc4")
			path.CreateAttr("stroke", "#9ca3af")
			path.CreateAttr("stroke-width", "0.25")
			continue
		}
		pct, ok := preferred[iso]
		var fill string
		if ok {
			fill = ColorForPct(pct)
		} else {
			fill = NoDataGray
		}
		path.CreateAttr("fill", fill)
		path.CreateAttr("stroke", "#4b5563")
		path.CreateAttr("stroke-width", "0.25")
	}

	return doc.WriteToBytes()
}

func clearPathPaint(path *etree.Element) {
	path.RemoveAttr("style")
	path.RemoveAttr("class")
	path.RemoveAttr("fill")
	path.RemoveAttr("stroke")
	path.RemoveAttr("stroke-width")
	path.RemoveAttr("fill-opacity")
	path.RemoveAttr("stroke-opacity")
	path.RemoveAttr("opacity")
}

func indexOfChild(parent, child *etree.Element) int {
	for i, c := range parent.ChildElements() {
		if c == child {
			return i
		}
	}
	return -1
}

func firstChildTag(parent *etree.Element, tag string) *etree.Element {
	for _, c := range parent.ChildElements() {
		if c.Tag == tag {
			return c
		}
	}
	return nil
}

func findByID(root *etree.Element, id string) *etree.Element {
	for _, el := range collectAny(root) {
		if el.SelectAttrValue("id", "") == id {
			return el
		}
	}
	return nil
}

func collectTag(root *etree.Element, tag string) []*etree.Element {
	var out []*etree.Element
	for _, el := range collectAny(root) {
		if el.Tag == tag {
			out = append(out, el)
		}
	}
	return out
}

func collectAny(root *etree.Element) []*etree.Element {
	var out []*etree.Element
	var walk func(*etree.Element)
	walk = func(el *etree.Element) {
		out = append(out, el)
		for _, c := range el.ChildElements() {
			walk(c)
		}
	}
	walk(root)
	return out
}

func childTextTitle(path *etree.Element) string {
	for _, c := range path.ChildElements() {
		if c.Tag == "title" {
			return normalizeTitle(c.Text())
		}
	}
	return ""
}
