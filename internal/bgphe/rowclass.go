package bgphe

import (
	"fmt"

	"github.com/pacific-monitor/pacific-monitor/internal/model"
)

// RowClass returns the CSS class for hybrid BGP/APNIC row coloring.
// HE rows: green when IPv6 routes are announced, red when v4-only (routes_v6 == 0).
// APNIC-only rows (no HE listing): green/red from ipv6preferred.
func RowClass(row model.BGPHENetworkRow) string {
	if !row.HERoutesNA {
		if row.RoutesV6 > 0 {
			return "bgphe-row--v6"
		}
		return "bgphe-row--pref-none"
	}
	if row.IPv6PreferredPct > 0 {
		return "bgphe-row--pref-ok"
	}
	return "bgphe-row--pref-none"
}

// RowAriaLabel describes the row for screen readers.
func RowAriaLabel(row model.BGPHENetworkRow) string {
	if !row.HERoutesNA {
		if row.RoutesV6 > 0 {
			return row.ASN + ", IPv6 routes announced in BGP"
		}
		if row.IPv6PreferredPct > 0 {
			return fmt.Sprintf("%s, IPv4 only in BGP (no IPv6 routes); APNIC IPv6 preferred %.2f%%", row.ASN, row.IPv6PreferredPct)
		}
		return row.ASN + ", IPv4 only in BGP, no IPv6 routes announced"
	}
	if row.IPv6PreferredPct > 0 {
		return fmt.Sprintf("%s, not on Hurricane Electric; IPv6 preferred %.2f%%", row.ASN, row.IPv6PreferredPct)
	}
	return row.ASN + ", not listed on Hurricane Electric; may not be connected to other ASNs in this economy"
}
