package ogmap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildMapPNG_smoke(t *testing.T) {
	root := findProjectRoot(t)
	svgPath := filepath.Join(root, "cmd/web/static/img/EEZ_Oceania.svg")
	svg, err := os.ReadFile(svgPath)
	if err != nil {
		t.Fatal(err)
	}
	idx := []byte(`{"countries":[{"iso2":"FJ","apnic_labs":{"preferred_pc_raw":42.5}}]}`)
	png, etag, err := BuildMapPNG(idx, svg)
	if err != nil {
		t.Fatal(err)
	}
	if len(png) < 1000 {
		t.Fatalf("png too small: %d", len(png))
	}
	if len(etag) != 64 {
		t.Fatalf("etag len: %d", len(etag))
	}
}

func TestColorForPct_matchesPctColorRampJS(t *testing.T) {
	// Goldens: same piecewise lerp + round as cmd/web/static/js/pct-color-ramp.js
	cases := []struct {
		p   float64
		hex string
	}{
		{0, "#ff0000"},
		{100, "#00ff00"},
		{25, "#e04a24"},
		{42.5, "#a78c23"},
		{62.5, "#6bb61f"},
	}
	for _, tc := range cases {
		got := ColorForPct(tc.p)
		if got != tc.hex {
			t.Fatalf("p=%v want %s got %s", tc.p, tc.hex, got)
		}
	}
}

func TestPreferredFromIndexJSON_paritiesWithMapHomeJS(t *testing.T) {
	// map-home.js only adds ISO when typeof preferred_pc_raw === 'number'
	nullIdx := []byte(`{"countries":[{"iso2":"FJ","apnic_labs":{"preferred_pc_raw":null}}]}`)
	m, err := PreferredFromIndexJSON(nullIdx)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m["FJ"]; ok {
		t.Fatal("null preferred_pc_raw must not produce a value (browser shows gray, not 0% red)")
	}
	numIdx := []byte(`{"countries":[{"iso2":"FJ","apnic_labs":{"preferred_pc_raw":0}}]}`)
	m, err = PreferredFromIndexJSON(numIdx)
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := m["FJ"]; !ok || v != 0 {
		t.Fatalf("explicit 0 must color like browser: got %+v", m)
	}
}

func TestSanitizeSVGColorsOKSVG_sevenDigitHex(t *testing.T) {
	in := []byte(`style="fill:#15b8cd6;stroke:#646464"`)
	out := string(SanitizeSVGColorsOKSVG(in))
	if strings.Contains(out, "#15b8cd6") {
		t.Fatalf("expected 7-digit hex shortened: %s", out)
	}
	if !strings.Contains(out, "#15b8cd") {
		t.Fatalf("expected #15b8cd: %s", out)
	}
}

func findProjectRoot(t *testing.T) string {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("go.mod not found")
	panic("unreachable")
}
