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
		{Label: "Lookup failed", ScoreRange: "grey", Description: "DNS timeout or non-NXDOMAIN error; use the MXToolbox link to re-check."},
		{Label: "p=none (monitoring)", ScoreRange: "25%", Description: "Record published; organizational policy is none."},
		{Label: "quarantine", ScoreRange: "75%", Description: "Stricter of p= and sp= is quarantine."},
		{Label: "reject", ScoreRange: "100%", Description: "Stricter of p= and sp= is reject."},
	}
}

// RPKIRampLegend returns RPKI column ramp semantics.
func RPKIRampLegend() []RampLegendItem {
	return []RampLegendItem{
		{Label: "Invalid route sample", ScoreRange: "0%", Description: "At least one sampled prefix failed RPKI validation (RIPEstat)."},
		{Label: "Partial / unknown", ScoreRange: "1–99%", Description: "Share of sampled prefixes with valid ROAs; unknown counts as non-valid."},
		{Label: "All sampled valid", ScoreRange: "100%", Description: "Every sampled prefix validated as RPKI-valid."},
		{Label: "No data", ScoreRange: "grey", Description: "RIPEstat error, no prefixes, or not yet collected."},
	}
}
