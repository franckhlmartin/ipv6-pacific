package ipv4outage

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/pacific-monitor/pacific-monitor/internal/httpserver"
)

var crawlerExemptPaths = map[string]struct{}{
	"/robots.txt":  {},
	"/sitemap.xml": {},
	"/og/map.png":  {},
}

// IsCrawlerExemptPath skips 566 for SEO/crawler assets.
func IsCrawlerExemptPath(path string) bool {
	if path == "" {
		path = "/"
	}
	_, ok := crawlerExemptPaths[path]
	return ok
}

func isLoopbackClient(r *http.Request) bool {
	ip := net.ParseIP(httpserver.RemoteIP(r))
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

func isIPv4Client(r *http.Request) bool {
	ip := net.ParseIP(httpserver.RemoteIP(r))
	if ip == nil {
		return true
	}
	return ip.To4() != nil
}

func allowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

// ShouldBlock reports whether to return 566 instead of invoking the next handler.
func ShouldBlock(r *http.Request, cfg Config, now time.Time) bool {
	if r == nil || !OutageActive(cfg, now) {
		return false
	}
	if !AppliesToHost(r, cfg) {
		return false
	}
	if !allowedMethod(r.Method) {
		return false
	}
	if IsCrawlerExemptPath(r.URL.Path) {
		return false
	}
	if isLoopbackClient(r) {
		return false
	}
	if !isIPv4Client(r) {
		return false
	}
	return true
}

// ParseRecoveryHeader extracts token from Retry-Over-IPv6-Recovery if present.
func ParseRecoveryHeader(r *http.Request) (recovered bool, token string) {
	raw := strings.TrimSpace(r.Header.Get("Retry-Over-IPv6-Recovery"))
	if raw == "" {
		return false, ""
	}
	lower := strings.ToLower(raw)
	if !strings.HasPrefix(lower, "recovered") {
		return false, ""
	}
	// recovered; token="abc"
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(part), "token=") {
			v := strings.TrimSpace(part[6:])
			v = strings.Trim(v, `"`)
			return true, v
		}
	}
	return true, ""
}
