package ripestat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// AnnouncedPrefixes returns up to maxPrefixes unique prefixes currently announced by the ASN.
func (c *Client) AnnouncedPrefixes(ctx context.Context, asn int, maxPrefixes int) ([]string, error) {
	if maxPrefixes <= 0 {
		maxPrefixes = 10
	}
	params := url.Values{}
	params.Set("resource", fmt.Sprintf("AS%d", asn))
	body, err := c.getJSON(ctx, "announced-prefixes", params)
	if err != nil {
		return nil, err
	}
	data, err := parseData(body)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Prefixes []struct {
			Prefix string `json:"prefix"`
		} `json:"prefixes"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	var out []string
	for _, p := range payload.Prefixes {
		prefix := strings.TrimSpace(p.Prefix)
		if prefix == "" {
			continue
		}
		if _, ok := seen[prefix]; ok {
			continue
		}
		seen[prefix] = struct{}{}
		out = append(out, prefix)
	}
	sort.Strings(out)
	if len(out) > maxPrefixes {
		out = out[:maxPrefixes]
	}
	return out, nil
}
