package ipv4outage

import (
	"net/http"
	"strings"
)

func stripPort(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if strings.HasPrefix(host, "[") {
		if i := strings.LastIndex(host, "]"); i >= 0 {
			rest := host[i+1:]
			if strings.HasPrefix(rest, ":") {
				return host[:i+1]
			}
		}
		return host
	}
	if i := strings.LastIndex(host, ":"); i >= 0 {
		return host[:i]
	}
	return host
}

func requestHost(r *http.Request) string {
	if r == nil {
		return ""
	}
	if xf := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); xf != "" {
		parts := strings.Split(xf, ",")
		return stripPort(strings.TrimSpace(parts[0]))
	}
	return stripPort(r.Host)
}

// AppliesToHost reports whether the request targets the configured main site hostname.
func AppliesToHost(r *http.Request, cfg Config) bool {
	want := strings.ToLower(strings.TrimSpace(cfg.OutageHost))
	if want == "" {
		return false
	}
	got := strings.ToLower(requestHost(r))
	return got == want
}
