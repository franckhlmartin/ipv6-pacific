package ogmap

import (
	"encoding/json"
	"math"
	"strings"
)

// PreferredFromIndexJSON builds iso2 → preferred_pc_raw like map-home.js buildPreferredByISO.
// Only economies with a numeric APNIC Labs preferred_pc_raw get a ramp color — same rule as the homepage EEZ map.
func PreferredFromIndexJSON(indexJSON []byte) (map[string]float64, error) {
	out := make(map[string]float64)
	if len(strings.TrimSpace(string(indexJSON))) == 0 {
		return out, nil
	}
	var payload struct {
		Countries []struct {
			ISO2      string `json:"iso2"`
			APNICLabs *struct {
				PreferredPcRaw *float64 `json:"preferred_pc_raw"`
			} `json:"apnic_labs"`
		} `json:"countries"`
	}
	if err := json.Unmarshal(indexJSON, &payload); err != nil {
		return nil, err
	}
	for _, row := range payload.Countries {
		iso := strings.ToUpper(strings.TrimSpace(row.ISO2))
		if iso == "" || row.APNICLabs == nil || row.APNICLabs.PreferredPcRaw == nil {
			continue
		}
		v := *row.APNICLabs.PreferredPcRaw
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		out[iso] = math.Max(0, math.Min(100, v))
	}
	return out, nil
}
