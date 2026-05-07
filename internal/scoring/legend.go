package scoring

// ScoreLegendMapping explains how a status maps to score points.
type ScoreLegendMapping struct {
	Label  string
	Points int
}

// ScoreLegendData provides plain-language, UI-safe score construction details.
type ScoreLegendData struct {
	RowRange       string
	RowFormula     string
	StatusMappings []ScoreLegendMapping
	DNSSECRule     string
	EconomyFormula string
}

// CountryScoreLegend returns score methodology aligned with RowScore and EconomyDeploymentScorePct.
func CountryScoreLegend() ScoreLegendData {
	return ScoreLegendData{
		RowRange:   "0-4 points",
		RowFormula: "Row score = DNS + Mail + Web + DNSSEC, where each column contributes 0 or 1 point.",
		StatusMappings: []ScoreLegendMapping{
			{Label: "IPv4-only", Points: 0},
			{Label: "Dual-stack", Points: 1},
			{Label: "IPv6-only", Points: 1},
			{Label: "Unknown", Points: 0},
		},
		DNSSECRule:     "DNSSEC uses signed=1 point; unsigned, error, and unknown=0 points.",
		EconomyFormula: "Economy deployment score (%) = average(RowScore/4) across domains x 100.",
	}
}
