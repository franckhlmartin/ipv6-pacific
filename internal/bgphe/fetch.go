// Package bgphe fetches and parses Hurricane Electric's public country BGP listing
// (bgp.he.net/country/{ISO2}). Data is scraped HTML; see model.BGPHETable for stored shape.
package bgphe

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pacific-monitor/pacific-monitor/internal/model"
)

const allowedHost = "bgp.he.net"

var browserUA = "Mozilla/5.0 (compatible; PacificIPv6Monitor/1.0; +https://github.com/pacific-monitor/pacific-monitor)"

// FetchCountryNetworks downloads the HE country page and returns parsed ASN rows.
// Only https://bgp.he.net/country/{ISO2} with ISO2 [A-Z]{2} is allowed (caller supplies ISO from config).
func FetchCountryNetworks(ctx context.Context, iso2 string, client *http.Client) (*model.BGPHETable, error) {
	if client == nil {
		client = http.DefaultClient
	}
	iso2 = strings.ToUpper(strings.TrimSpace(iso2))
	if len(iso2) != 2 || iso2[0] < 'A' || iso2[0] > 'Z' || iso2[1] < 'A' || iso2[1] > 'Z' {
		return nil, fmt.Errorf("bgphe: invalid iso2 %q", iso2)
	}
	rawURL := fmt.Sprintf("https://bgp.he.net/country/%s", iso2)
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "https" || parsed.Hostname() != allowedHost || parsed.Path != "/country/"+iso2 {
		return nil, errors.New("bgphe: url not allowlisted")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", browserUA)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bgphe: http %d", resp.StatusCode)
	}

	networks, err := ParseCountryNetworksHTML(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	return &model.BGPHETable{
		SourceURL: rawURL,
		FetchedAt: time.Now().UTC(),
		Networks:  networks,
	}, nil
}
