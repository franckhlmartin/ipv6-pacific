package bgphe

import (
	"testing"

	"github.com/pacific-monitor/pacific-monitor/internal/model"
)

func TestRowClass(t *testing.T) {
	tests := []struct {
		name string
		row  model.BGPHENetworkRow
		want string
	}{
		{"v6 routes", model.BGPHENetworkRow{RoutesV6: 3, IPv6PreferredPct: 0}, "bgphe-row--v6"},
		{"v4 only zero pref", model.BGPHENetworkRow{RoutesV6: 0, IPv6PreferredPct: 0}, "bgphe-row--pref-none"},
		{"v4 only with pref", model.BGPHENetworkRow{RoutesV6: 0, IPv6PreferredPct: 5}, "bgphe-row--pref-ok"},
		{"apnic only pref", model.BGPHENetworkRow{HERoutesNA: true, IPv6PreferredPct: 88.89}, "bgphe-row--pref-ok"},
		{"apnic only no pref", model.BGPHENetworkRow{HERoutesNA: true, IPv6PreferredPct: 0}, "bgphe-row--pref-none"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := RowClass(tc.row); got != tc.want {
				t.Fatalf("RowClass() = %q, want %q", got, tc.want)
			}
		})
	}
}
