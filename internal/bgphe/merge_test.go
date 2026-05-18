package bgphe

import (
	"testing"
	"time"

	"github.com/pacific-monitor/pacific-monitor/internal/apnicstats"
	"github.com/pacific-monitor/pacific-monitor/internal/model"
)

func TestMergeWithAPNICPreferred_tk(t *testing.T) {
	he := &model.BGPHETable{
		SourceURL: "https://bgp.he.net/country/TK",
		FetchedAt: time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC),
		Networks: []model.BGPHENetworkRow{
			{ASN: "AS55523", ASNNumber: 55523, Name: "Teletok", RoutesV4: 5, RoutesV6: 0},
		},
	}
	apnic := &apnicstats.FetchResult{
		SourceURL: "https://stats.labs.apnic.net/ipv6/TK",
		FetchedAt: time.Date(2026, 5, 18, 12, 1, 0, 0, time.UTC),
		Rows: []apnicstats.ASNRow{
			{ASN: "AS14593", ASNNumber: 14593, Name: "Starlink", IPv6PreferredPct: 88.89, Samples: 108},
			{ASN: "AS55523", ASNNumber: 55523, Name: "Teletok APNIC", IPv6PreferredPct: 0, Samples: 106},
		},
	}

	merged := MergeWithAPNICPreferred(he, apnic)
	if merged == nil {
		t.Fatal("nil merged")
	}
	if len(merged.Networks) != 2 {
		t.Fatalf("want 2 networks, got %d", len(merged.Networks))
	}
	if merged.Networks[0].ASN != "AS14593" || !merged.Networks[0].HERoutesNA {
		t.Fatalf("row0 want APNIC-only Starlink first: %+v", merged.Networks[0])
	}
	if merged.Networks[0].IPv6PreferredPct != 88.89 {
		t.Fatalf("row0 preferred: %v", merged.Networks[0].IPv6PreferredPct)
	}
	row1 := merged.Networks[1]
	if row1.ASN != "AS55523" || row1.HERoutesNA {
		t.Fatalf("row1 want HE Teletok: %+v", row1)
	}
	if row1.IPv6PreferredPct != 0 || row1.RoutesV6 != 0 {
		t.Fatalf("row1 routes/preferred: v6=%d pref=%v", row1.RoutesV6, row1.IPv6PreferredPct)
	}
}

func TestMergeWithAPNICPreferred_apnicOnly(t *testing.T) {
	apnic := &apnicstats.FetchResult{
		SourceURL: "https://stats.labs.apnic.net/ipv6/TK",
		FetchedAt: time.Now().UTC(),
		Rows: []apnicstats.ASNRow{
			{ASN: "AS14593", ASNNumber: 14593, Name: "Starlink", IPv6PreferredPct: 50, Samples: 10},
		},
	}
	merged := MergeWithAPNICPreferred(nil, apnic)
	if len(merged.Networks) != 1 || !merged.Networks[0].HERoutesNA {
		t.Fatalf("got %+v", merged.Networks)
	}
}
