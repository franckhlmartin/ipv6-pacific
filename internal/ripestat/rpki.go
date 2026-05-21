package ripestat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pacific-monitor/pacific-monitor/internal/model"
	"github.com/pacific-monitor/pacific-monitor/internal/rampscore"
)

// Validation status strings from RIPEstat rpki-validation endpoint.
const (
	StatusValid         = "valid"
	StatusInvalidASN    = "invalid_asn"
	StatusInvalidLength = "invalid_length"
	StatusUnknown       = "unknown"
)

// ValidatePrefix checks RPKI validity for (asn, prefix).
func (c *Client) ValidatePrefix(ctx context.Context, asn int, prefix string) (status string, err error) {
	params := url.Values{}
	params.Set("resource", fmt.Sprintf("%d", asn))
	params.Set("prefix", prefix)
	body, err := c.getJSON(ctx, "rpki-validation", params)
	if err != nil {
		return "", err
	}
	data, err := parseData(body)
	if err != nil {
		return "", err
	}
	var row struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(data, &row); err != nil {
		return "", err
	}
	return strings.ToLower(strings.TrimSpace(row.Status)), nil
}

// ASNResult holds aggregated RPKI sampling for one ASN.
type ASNResult struct {
	CheckedPrefixes int
	Valid           int
	Invalid         int
	Unknown         int
	ScorePct        float64
	WorstStatus     string
	Error           string
	CheckedAt       time.Time
}

// EnrichASN samples prefixes and fills RPKI fields on row.
func (c *Client) EnrichASN(ctx context.Context, row *model.BGPHENetworkRow, maxPrefixes int) {
	if row == nil || row.ASNNumber <= 0 {
		return
	}
	prefixes, err := c.AnnouncedPrefixes(ctx, row.ASNNumber, maxPrefixes)
	if err != nil {
		row.RPKIError = err.Error()
		return
	}
	if len(prefixes) == 0 {
		row.RPKIError = "no announced prefixes"
		return
	}
	var valid, invalid, unknown int
	for _, prefix := range prefixes {
		st, err := c.ValidatePrefix(ctx, row.ASNNumber, prefix)
		if err != nil {
			row.RPKIError = err.Error()
			return
		}
		switch st {
		case StatusValid:
			valid++
		case StatusInvalidASN, StatusInvalidLength:
			invalid++
		default:
			unknown++
		}
	}
	row.RPKICheckedPrefixes = len(prefixes)
	row.RPKIValid = valid
	row.RPKIInvalid = invalid
	row.RPKIUnknown = unknown
	row.RPKIWorstStatus = rampscore.RPKIWorstStatus(valid, invalid, unknown)
	row.RPKIScorePct, _ = rampscore.RPKIScorePct(valid, invalid, unknown, len(prefixes), "")
	row.RPKISourceURL = "https://stat.ripe.net/docs/data-api/api-endpoints/rpki-validation"
	row.RPKICheckedAt = time.Now().UTC()
	row.RPKIError = ""
}
