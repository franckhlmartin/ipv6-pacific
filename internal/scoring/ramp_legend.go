package scoring

// RampLegendItem documents a 0–100% ramp band used by DMARC and RPKI columns.
type RampLegendItem struct {
	Label       string
	ScoreRange  string
	Description string
}

// DMARCRampLegend returns DMARC column ramp semantics.
func DMARCRampLegend() []RampLegendItem {
	return []RampLegendItem{
		{Label: "No DMARC record", ScoreRange: "0%", Description: "No _dmarc TXT at the domain apex (NXDOMAIN or no v=DMARC1 in answers)."},
		{Label: "Lookup failed", ScoreRange: "grey", Description: "DNS timeout or non-NXDOMAIN error; use the dmarcian link to re-check."},
		{Label: "p=none (monitoring)", ScoreRange: "25%", Description: "Record published; organizational policy is none."},
		{Label: "quarantine", ScoreRange: "75%", Description: "Stricter of p= and sp= is quarantine."},
		{Label: "reject", ScoreRange: "100%", Description: "Stricter of p= and sp= is reject."},
	}
}

// RPKIRampLegend returns RPKI column ramp semantics.
func RPKIRampLegend() []RampLegendItem {
	return []RampLegendItem{
		{Label: "RPKI %", ScoreRange: "0–100%", Description: "Share of sampled prefixes RIPEstat marked valid (e.g. 8/10 → 80%). Invalid and unknown prefixes lower the percentage."},
		{Label: "RPKI status", ScoreRange: "text", Description: "Worst result in the sample (valid, invalid, or unknown) with counts — use the link for RIPEstat details."},
		{Label: "No data", ScoreRange: "grey", Description: "RIPEstat error, no prefixes, or not yet collected."},
	}
}
