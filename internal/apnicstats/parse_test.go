package apnicstats

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCountryASNTableHTML_tkFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "tk_drawtable_fragment.html"))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := ParseCountryASNTableHTML(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(rows))
	}
	if rows[0].ASN != "AS14593" || rows[0].ASNNumber != 14593 {
		t.Fatalf("row0 asn: %+v", rows[0])
	}
	if rows[0].IPv6PreferredPct != 88.89 {
		t.Fatalf("row0 preferred: got %v", rows[0].IPv6PreferredPct)
	}
	if rows[0].Samples != 108 {
		t.Fatalf("row0 samples: %d", rows[0].Samples)
	}
	if rows[1].ASN != "AS55523" || rows[1].IPv6PreferredPct != 0 {
		t.Fatalf("row1: %+v", rows[1])
	}
}
