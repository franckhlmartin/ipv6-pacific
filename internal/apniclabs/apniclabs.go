package apniclabs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pacific-monitor/pacific-monitor/internal/model"
)

const allowedHost = "data1.labs.apnic.net"

// FetchLatest pulls v6economy/{CC}.json and returns the chronologically latest raw sample.
func FetchLatest(ctx context.Context, iso2 string, client *http.Client) (*model.APNICSnapshot, error) {
	iso2 = strings.ToUpper(strings.TrimSpace(iso2))
	if len(iso2) != 2 {
		return nil, fmt.Errorf("invalid iso2: %q", iso2)
	}
	u := fmt.Sprintf("https://data1.labs.apnic.net/v6stats/v6economy/%s.json", iso2)
	parsed, err := url.Parse(u)
	if err != nil {
		return nil, err
	}
	if parsed.Hostname() != allowedHost || parsed.Scheme != "https" {
		return nil, errors.New("apniclabs: host not allowlisted")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 50<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("apniclabs: http %d", resp.StatusCode)
	}

	var wrap struct {
		Copyright string          `json:"copyright"`
		Data      json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		return nil, err
	}

	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(wrap.Data, &rows); err != nil {
		return nil, err
	}

	var latestDate string
	var latestRow map[string]json.RawMessage
	for _, row := range rows {
		var date string
		if d, ok := row["date"]; ok {
			_ = json.Unmarshal(d, &date)
		}
		if date > latestDate {
			latestDate = date
			latestRow = row
		}
	}
	if latestRow == nil {
		return nil, fmt.Errorf("apniclabs: empty data for %s", iso2)
	}

	var updated string
	_ = json.Unmarshal(latestRow["updated"], &updated)

	var raw struct {
		Seen         float64 `json:"seen"`
		Preferred    float64 `json:"preferred"`
		PreferredPct float64 `json:"preferred_pc"`
	}
	if r, ok := latestRow["raw"]; ok {
		_ = json.Unmarshal(r, &raw)
	}

	return &model.APNICSnapshot{
		Copyright:    wrap.Copyright,
		SourceURL:    u,
		Date:         latestDate,
		Updated:      updated,
		Preferred:    raw.Preferred,
		PreferredPct: raw.PreferredPct,
		Seen:         raw.Seen,
		FetchedAt:    time.Now().UTC(),
	}, nil
}
