package bgphe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCountryNetworksHTML_fixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "country_fj_fragment.html"))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := ParseCountryNetworksHTML(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("want 3 rows, got %d", len(rows))
	}
	if rows[0].ASN != "AS38442" || rows[0].ASNNumber != 38442 || rows[0].Name != "Vodafone Fiji Limited" {
		t.Fatalf("row0: %+v", rows[0])
	}
	if rows[0].RoutesV4 != 63 || rows[0].RoutesV6 != 19 {
		t.Fatalf("row0 routes: v4=%d v6=%d", rows[0].RoutesV4, rows[0].RoutesV6)
	}
	if rows[1].ASN != "AS9241" || rows[1].RoutesV6 != 0 {
		t.Fatalf("row1: %+v", rows[1])
	}
	if rows[2].RoutesV6 != 2 {
		t.Fatalf("row2 v6: %+v", rows[2])
	}
}
