package bgphe

import (
	"fmt"

	"github.com/pacific-monitor/pacific-monitor/internal/model"
)

// RowClass returns the CSS class for hybrid BGP/APNIC row coloring.
func RowClass(row model.BGPHENetworkRow) string {
	if !row.HERoutesNA && row.RoutesV6 > 0 {
		return "bgphe-row--v6"
	}
	if row.IPv6PreferredPct > 0 {
		return "bgphe-row--pref-ok"
	}
	return "bgphe-row--pref-none"
}

// RowAriaLabel describes the row for screen readers.
func RowAriaLabel(row model.BGPHENetworkRow) string {
	if !row.HERoutesNA && row.RoutesV6 > 0 {
		return row.ASN + ", IPv6 routes announced in BGP"
	}
	if row.IPv6PreferredPct > 0 {
		return fmt.Sprintf("%s, IPv6 preferred %.2f%%", row.ASN, row.IPv6PreferredPct)
	}
	if row.HERoutesNA {
		return row.ASN + ", not listed on Hurricane Electric; may not be connected to other ASNs in this economy"
	}
	return row.ASN + ", no IPv6 routes and no IPv6 preference measured"
}
