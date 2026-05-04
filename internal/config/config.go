package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// PacificCountry is one economy entry from config/pacific_iso2.yaml.
type PacificCountry struct {
	ISO2         string    `yaml:"iso2"`
	Name         string    `yaml:"name"`
	Centroid     []float64 `yaml:"centroid"`                // [lon, lat]
	ExcludeAPNIC bool      `yaml:"exclude_apnic,omitempty"` // AU/NZ style skip for Labs fetch
}

// PacificList wraps the YAML root.
type PacificList struct {
	Countries []PacificCountry `yaml:"countries"`
}

// DomainEntry is one monitored apex domain.
type DomainEntry struct {
	Domain       string `yaml:"domain"`
	Organization string `yaml:"organization,omitempty"`
	Sector       string `yaml:"sector,omitempty"`
	Regional     bool   `yaml:"regional,omitempty"`
	// WebURL is tried first for HTTPS reachability when the public site is not at apex or www (e.g. gov.fj → https://www.itc.gov.fj/).
	WebURL string `yaml:"web_url,omitempty"`
}

// DomainsFile is config/domains/{ISO}.yaml body.
type DomainsFile struct {
	Domains []DomainEntry `yaml:"domains"`
}

// LoadPacific reads config/pacific_iso2.yaml from repo root or dir.
func LoadPacific(dir string) (*PacificList, error) {
	path := filepath.Join(dir, "config", "pacific_iso2.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var list PacificList
	if err := yaml.Unmarshal(raw, &list); err != nil {
		return nil, err
	}
	return &list, nil
}

// LoadDomainsFile loads a single country YAML.
func LoadDomainsFile(path string) (*DomainsFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var df DomainsFile
	if err := yaml.Unmarshal(raw, &df); err != nil {
		return nil, err
	}
	for i := range df.Domains {
		df.Domains[i].Domain = strings.TrimSpace(strings.ToLower(df.Domains[i].Domain))
		df.Domains[i].WebURL = strings.TrimSpace(df.Domains[i].WebURL)
	}
	return &df, nil
}

// AllowedISO builds a map of uppercase ISO2 codes present in pacific_iso2.
func AllowedISO(list *PacificList) map[string]struct{} {
	m := make(map[string]struct{})
	if list == nil {
		return m
	}
	for _, c := range list.Countries {
		m[strings.ToUpper(c.ISO2)] = struct{}{}
	}
	return m
}

// ValidateISO returns error if iso2 not in allowlist.
func ValidateISO(allowed map[string]struct{}, iso2 string) error {
	u := strings.ToUpper(strings.TrimSpace(iso2))
	if len(u) != 2 {
		return fmt.Errorf("invalid iso2")
	}
	if _, ok := allowed[u]; !ok {
		return fmt.Errorf("iso2 not allowed: %s", u)
	}
	return nil
}
