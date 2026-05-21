package checks

import "github.com/pacific-monitor/pacific-monitor/internal/model"

// LegendStatusItem explains one shared cell color/status meaning.
type LegendStatusItem struct {
	Class       string
	Label       string
	Description string
	Points      int
}

// LegendCheckExplanation describes how a checker's compact display should be read.
type LegendCheckExplanation struct {
	ID           string
	Title        string
	Format       string
	PlainMeaning string
	Notes        []string
}

// LegendLocationItem explains compact location tags appended in DNS/Mail/Web cells.
type LegendLocationItem struct {
	Tag         string
	Label       string
	Description string
}

// CountryLegendStatusItems returns shared status semantics used by DNS/Mail/Web/DNSSEC cell colors.
func CountryLegendStatusItems() []LegendStatusItem {
	return []LegendStatusItem{
		{
			Class:       string(model.DeployIPv4Only),
			Label:       "IPv4-only",
			Description: "No IPv6 operational path observed for this test result.",
			Points:      0,
		},
		{
			Class:       string(model.DeployDual),
			Label:       "Dual-stack",
			Description: "Both IPv4 and IPv6 paths are present; IPv6 is available with IPv4.",
			Points:      1,
		},
		{
			Class:       string(model.DeployIPv6Only),
			Label:       "IPv6-only",
			Description: "IPv6 path is operational for this test result.",
			Points:      1,
		},
		{
			Class:       string(model.DeployUnknown),
			Label:       "Unknown",
			Description: "Measurement was not conclusive or the test was not available.",
			Points:      0,
		},
	}
}

// CountryLegendCheckExplanations returns per-check legend entries.
func CountryLegendCheckExplanations() []LegendCheckExplanation {
	return []LegendCheckExplanation{
		dnsLegendExplanation(),
		mailLegendExplanation(),
		webLegendExplanation(),
		dnssecLegendExplanation(),
		dmarcLegendExplanation(),
	}
}

// CountryLegendLocationItems returns the location-key used in DNS/Mail/Web display strings.
func CountryLegendLocationItems() []LegendLocationItem {
	return []LegendLocationItem{
		{Tag: "I", Label: "Inside", Description: "Service hostname is the tested domain or a subdomain under it."},
		{Tag: "P", Label: "Parent-related", Description: "Service hostname resolves via a parent/suffix relationship to the tested domain."},
		{Tag: "O", Label: "Outside", Description: "Service hostname is outside the tested domain suffix."},
		{Tag: "M", Label: "Mixed", Description: "Multiple service hosts span more than one location class."},
		{Tag: "-", Label: "Not available", Description: "No location could be derived from the observed data."},
	}
}
