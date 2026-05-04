package model

import "time"

// CollectorVersion is embedded in generated JSON for traceability.
const CollectorVersion = "0.1.0"

// DeployClass matches UI color semantics (orange / blue / green).
type DeployClass string

const (
	DeployIPv4Only DeployClass = "ipv4_only"
	DeployDual     DeployClass = "dual_stack"
	DeployIPv6Only DeployClass = "ipv6_only"
	DeployUnknown  DeployClass = "unknown"
)

// Index is written to data/index.json.
type Index struct {
	GeneratedAt      time.Time      `json:"generated_at"`
	CollectorVersion string         `json:"collector_version"`
	Countries        []IndexCountry `json:"countries"`
}

// IndexCountry summarizes one economy for the map and listing APIs.
type IndexCountry struct {
	ISO2          string            `json:"iso2"`
	Name          string            `json:"name"`
	DomainCount   int               `json:"domain_count"`
	Summary       DeploymentSummary `json:"summary"`
	Centroid      []float64         `json:"centroid,omitempty"` // [lon, lat] for GeoJSON/map
	APNICLabs     *APNICSnapshot    `json:"apnic_labs,omitempty"`
	LastCollected *time.Time        `json:"last_collected,omitempty"`
}

// DeploymentSummary rolls up domain deployment classes for counters on /country header.
type DeploymentSummary struct {
	IPv4Only int `json:"ipv4_only"`
	Dual     int `json:"dual_stack"`
	IPv6Only int `json:"ipv6_only"`
	DNS      int `json:"dns_measured"` // domains with DNS column populated
	Mail     int `json:"mail_measured"`
	Web      int `json:"web_measured"`
	DNSSEC   int `json:"dnssec_measured"`
}

// CountryFile is data/countries/{ISO2}.json.
type CountryFile struct {
	ISO2             string         `json:"iso2"`
	Name             string         `json:"name"`
	GeneratedAt      time.Time      `json:"generated_at"` // UTC wall time when this file was last written
	CollectorVersion string         `json:"collector_version"`
	Domains          []DomainResult `json:"domains"`
	APNICLabs        *APNICSnapshot `json:"apnic_labs,omitempty"`
}

// DomainResult is one row in the Afrinic-style table.
type DomainResult struct {
	Domain       string        `json:"domain"`
	Organization string        `json:"organization,omitempty"`
	Sector       string        `json:"sector,omitempty"`
	DNS          ServiceColumn `json:"dns"`
	Mail         ServiceColumn `json:"mail"`
	Web          ServiceColumn `json:"web"`
	DNSSEC       DNSSECColumn  `json:"dnssec"`
	RollupClass  DeployClass   `json:"rollup_class"`
	Error        string        `json:"error,omitempty"`
	CollectedAt  time.Time     `json:"collected_at,omitempty"` // UTC when this row's checks finished
}

// ServiceMetrics holds NIST-style counts per address family.
type ServiceMetrics struct {
	Configured  int `json:"configured"`
	Reachable   int `json:"reachable"`
	Operational int `json:"operational"`
}

// ServiceColumn combines v4/v6 metrics + location tag + display string for table cells.
type ServiceColumn struct {
	Location        string         `json:"location"` // I, P, O, M, -, S, L
	IPv4            ServiceMetrics `json:"ipv4"`
	IPv6            ServiceMetrics `json:"ipv6"`
	Class           DeployClass    `json:"class"`
	Display         string         `json:"display"` // compact summary for UI
	IntentionallyNA bool           `json:"intentionally_na,omitempty"`
}

// DNSSECColumn stores a simplified DNSSEC state for the zone apex.
type DNSSECColumn struct {
	State   string `json:"state"`   // good, island, unsigned, error
	Summary string `json:"summary"` // e.g. S/V/C shorthand when applicable
	Display string `json:"display"`
}

// APNICSnapshot is merged from v6economy/{CC}.json latest row.
type APNICSnapshot struct {
	Copyright    string    `json:"copyright,omitempty"`
	SourceURL    string    `json:"source_url"`
	Date         string    `json:"date"` // measurement date YYYY-MM-DD
	Updated      string    `json:"updated,omitempty"`
	Preferred    float64   `json:"preferred_raw,omitempty"`
	PreferredPct float64   `json:"preferred_pc_raw,omitempty"`
	Seen         float64   `json:"seen_raw,omitempty"`
	FetchedAt    time.Time `json:"fetched_at"`
}
