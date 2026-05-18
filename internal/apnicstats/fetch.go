// Package apnicstats fetches and parses APNIC Labs per-ASN IPv6 stats from stats.labs.apnic.net/ipv6/{ISO2}.
package apnicstats

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const allowedHost = "stats.labs.apnic.net"

var browserUA = "Mozilla/5.0 (compatible; PacificIPv6Monitor/1.0; +https://github.com/pacific-monitor/pacific-monitor)"

// FetchResult holds parsed ASN rows and fetch metadata.
type FetchResult struct {
	SourceURL string
	FetchedAt time.Time
	Rows      []ASNRow
}

// FetchCountryASNTable downloads the APNIC Labs country stats page and parses the ASN table.
func FetchCountryASNTable(ctx context.Context, iso2 string, client *http.Client) (*FetchResult, error) {
	if client == nil {
		client = http.DefaultClient
	}
	iso2 = strings.ToUpper(strings.TrimSpace(iso2))
	if len(iso2) != 2 || iso2[0] < 'A' || iso2[0] > 'Z' || iso2[1] < 'A' || iso2[1] > 'Z' {
		return nil, fmt.Errorf("apnicstats: invalid iso2 %q", iso2)
	}
	rawURL := fmt.Sprintf("https://stats.labs.apnic.net/ipv6/%s", iso2)
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "https" || parsed.Hostname() != allowedHost || parsed.Path != "/ipv6/"+iso2 {
		return nil, errors.New("apnicstats: url not allowlisted")
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

	body, err := io.ReadAll(io.LimitReader(resp.Body, 15<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("apnicstats: http %d", resp.StatusCode)
	}

	rows, err := ParseCountryASNTableHTMLString(string(body))
	if err != nil {
		return nil, err
	}

	return &FetchResult{
		SourceURL: rawURL,
		FetchedAt: time.Now().UTC(),
		Rows:      rows,
	}, nil
}
