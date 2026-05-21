package model

import "time"

// CollectorVersion is embedded in generated JSON for traceability.
const CollectorVersion = "0.3.0"

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
	ISO2               string            `json:"iso2"`
	Name               string            `json:"name"`
	DomainCount        int               `json:"domain_count"`
	Summary            DeploymentSummary `json:"summary"`
	DeploymentScorePct float64           `json:"deployment_score_pct"` // mean RowScore / 4 · 100 (0–100); comparable across economies
	Centroid           []float64         `json:"centroid,omitempty"`   // [lon, lat] for GeoJSON/map
	APNICLabs          *APNICSnapshot    `json:"apnic_labs,omitempty"`
	LastCollected      *time.Time        `json:"last_collected,omitempty"`
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
	ISO2                 string         `json:"iso2"`
	Name                 string         `json:"name"`
	GeneratedAt          time.Time      `json:"generated_at"` // UTC wall time when this file was last written
	CollectorVersion     string         `json:"collector_version"`
	Domains              []DomainResult `json:"domains"`
	APNICLabs            *APNICSnapshot `json:"apnic_labs,omitempty"`
	BGPHurricaneElectric *BGPHETable    `json:"bgp_he_net,omitempty"` // Hurricane Electric country BGP listing (scraped)
}

// BGPHETable is a merged snapshot: HE country BGP listing plus APNIC Labs per-ASN IPv6 preference.
type BGPHETable struct {
	SourceURL            string            `json:"source_url"`
	FetchedAt            time.Time         `json:"fetched_at"`
	APNICStatsSourceURL  string            `json:"apnic_stats_source_url,omitempty"`
	APNICStatsFetchedAt  time.Time         `json:"apnic_stats_fetched_at,omitempty"`
	Networks             []BGPHENetworkRow `json:"networks"`
}

// BGPHENetworkRow is one ASN in the merged BGP / APNIC table.
type BGPHENetworkRow struct {
	ASN              string  `json:"asn"`        // e.g. AS38442
	ASNNumber        int     `json:"asn_number"` // numeric ASN for sorting
	Name             string  `json:"name"`
	RoutesV4         int     `json:"routes_v4"`
	RoutesV6         int     `json:"routes_v6"`
	IPv6PreferredPct float64 `json:"ipv6preferred"` // APNIC Labs IPv6 preferred % (30-day smoothed)
	HERoutesNA       bool    `json:"he_routes_na,omitempty"`
	APNICSamples     int     `json:"apnic_samples,omitempty"`
	// RPKI (RIPEstat sampled prefix validation)
	RPKICheckedPrefixes int       `json:"rpki_checked_prefixes,omitempty"`
	RPKIValid           int       `json:"rpki_valid,omitempty"`
	RPKIInvalid         int       `json:"rpki_invalid,omitempty"`
	RPKIUnknown         int       `json:"rpki_unknown,omitempty"`
	RPKIScorePct        float64   `json:"rpki_score_pct,omitempty"`
	RPKIWorstStatus     string    `json:"rpki_worst_status,omitempty"` // valid, invalid, unknown
	RPKIError           string    `json:"rpki_error,omitempty"`
	RPKISourceURL       string    `json:"rpki_source_url,omitempty"`
	RPKICheckedAt       time.Time `json:"rpki_checked_at,omitempty"`
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
	DMARC        DMARCColumn   `json:"dmarc"`
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

// DMARCColumn stores DMARC policy from _dmarc.{apex} TXT.
type DMARCColumn struct {
	State           string  `json:"state"` // absent, present, error
	Exists          bool    `json:"exists"`
	Policy          string  `json:"policy,omitempty"`           // effective org: none, quarantine, reject
	SubdomainPolicy string  `json:"subdomain_policy,omitempty"` // effective sp after inherit
	RawP            string  `json:"raw_p,omitempty"`
	RawSP           string  `json:"raw_sp,omitempty"`
	Display         string  `json:"display"`
	ScorePct        float64 `json:"score_pct,omitempty"` // 0-100 for UI ramp; absent=0, error=omit
}

// APNICSnapshot is merged from v6economy/{CC}.json latest row.
type APNICSnapshot struct {
	Copyright string  `json:"copyright,omitempty"`
	SourceURL string  `json:"source_url"`
	Date      string  `json:"date"` // measurement date YYYY-MM-DD
	Updated   string  `json:"updated,omitempty"`
	Preferred float64 `json:"preferred_raw,omitempty"`
	// No omitempty: 0% is valid data; omitting the field made clients treat it like “no APNIC”.
	PreferredPct float64   `json:"preferred_pc_raw"`
	Seen         float64   `json:"seen_raw,omitempty"`
	FetchedAt    time.Time `json:"fetched_at"`
}
