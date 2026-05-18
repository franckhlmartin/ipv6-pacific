package bgphe

import (
	"sort"
	"time"

	"github.com/pacific-monitor/pacific-monitor/internal/apnicstats"
	"github.com/pacific-monitor/pacific-monitor/internal/model"
)

// MergeWithAPNICPreferred combines HE BGP rows with APNIC Labs per-ASN stats.
// APNIC-only ASNs are appended with HERoutesNA set; output is sorted by APNIC samples descending.
func MergeWithAPNICPreferred(he *model.BGPHETable, apnic *apnicstats.FetchResult) *model.BGPHETable {
	if apnic == nil || len(apnic.Rows) == 0 {
		return he
	}

	apnicBy := make(map[int]apnicstats.ASNRow, len(apnic.Rows))
	for _, r := range apnic.Rows {
		apnicBy[r.ASNNumber] = r
	}

	var out model.BGPHETable
	if he != nil {
		out.SourceURL = he.SourceURL
		out.FetchedAt = he.FetchedAt
	}
	out.APNICStatsSourceURL = apnic.SourceURL
	out.APNICStatsFetchedAt = apnic.FetchedAt

	seen := make(map[int]struct{})
	if he != nil {
		for _, row := range he.Networks {
			merged := row
			if ar, ok := apnicBy[row.ASNNumber]; ok {
				merged.IPv6PreferredPct = ar.IPv6PreferredPct
				merged.APNICSamples = ar.Samples
				if merged.Name == "" {
					merged.Name = ar.Name
				}
			}
			out.Networks = append(out.Networks, merged)
			seen[row.ASNNumber] = struct{}{}
		}
	}

	for _, ar := range apnic.Rows {
		if _, ok := seen[ar.ASNNumber]; ok {
			continue
		}
		out.Networks = append(out.Networks, model.BGPHENetworkRow{
			ASN:              ar.ASN,
			ASNNumber:        ar.ASNNumber,
			Name:             ar.Name,
			IPv6PreferredPct: ar.IPv6PreferredPct,
			APNICSamples:     ar.Samples,
			HERoutesNA:       true,
		})
	}

	sort.Slice(out.Networks, func(i, j int) bool {
		si, sj := out.Networks[i].APNICSamples, out.Networks[j].APNICSamples
		if si != sj {
			return si > sj
		}
		return out.Networks[i].ASNNumber < out.Networks[j].ASNNumber
	})

	if out.FetchedAt.IsZero() {
		out.FetchedAt = time.Now().UTC()
	}
	return &out
}
