package ipv4outage

import (
	"net/http"
	"strconv"
	"strings"
)

// PrefersProblemJSON reports whether the response body should be RFC 9457 JSON.
func PrefersProblemJSON(r *http.Request) bool {
	if r == nil {
		return false
	}
	path := r.URL.Path
	if strings.HasPrefix(path, "/api/") {
		return true
	}
	return acceptPrefersJSON(r.Header.Get("Accept"))
}

func acceptPrefersJSON(accept string) bool {
	accept = strings.TrimSpace(accept)
	if accept == "" {
		return false
	}
	bestJSON := -1.0
	bestOther := -1.0
	for _, part := range strings.Split(accept, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		media := part
		q := 1.0
		if i := strings.Index(part, ";"); i >= 0 {
			media = strings.TrimSpace(part[:i])
			params := part[i+1:]
			pl := strings.TrimSpace(strings.ToLower(params))
			if strings.HasPrefix(pl, "q=") {
				if parsed, err := strconv.ParseFloat(strings.TrimSpace(pl[2:]), 64); err == nil && parsed >= 0 && parsed <= 1 {
					q = parsed
				}
			}
		}
		media = strings.ToLower(media)
		switch media {
		case "application/problem+json", "application/json":
			if q > bestJSON {
				bestJSON = q
			}
		case "*/*":
			if q > bestOther {
				bestOther = q
			}
		default:
			if strings.HasPrefix(media, "text/") && q > bestOther {
				bestOther = q
			}
		}
	}
	if bestJSON < 0 {
		return false
	}
	return bestJSON >= bestOther
}
