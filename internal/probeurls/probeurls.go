package probeurls

import (
	"net"
	"net/url"
	"os"
	"strings"
)

const (
	DefaultHost = "pacific.ipv6forum.com"
	healthzPath = "/api/healthz"
)

// Config holds resolved probe healthz URLs for templates, embed bundles, and CSP.
type Config struct {
	V4 string
	V6 string
	DS string
}

// Load reads PROBE_* from the environment. Unset values use defaults derived from
// PUBLIC_SITE_URL or DefaultHost (https://ipv4.<host>, https://ipv6.<host>, https://<host>).
func Load() Config {
	host := SiteHost()
	return Config{
		V4: envOrDefault("PROBE_V4_URL", "https://ipv4."+host+healthzPath),
		V6: envOrDefault("PROBE_V6_URL", "https://ipv6."+host+healthzPath),
		DS: envOrDefault("PROBE_DS_URL", "https://"+host+healthzPath),
	}
}

// SiteHost returns the dual-stack site hostname for default probe URLs.
func SiteHost() string {
	if h := hostFromPublicSiteURL(os.Getenv("PUBLIC_SITE_URL")); h != "" {
		return h
	}
	return DefaultHost
}

// Origins returns unique scheme://host tokens for CSP connect-src.
func Origins(cfg Config) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, raw := range []string{cfg.V4, cfg.V6, cfg.DS} {
		o := origin(raw)
		if o == "" {
			continue
		}
		if _, ok := seen[o]; ok {
			continue
		}
		seen[o] = struct{}{}
		out = append(out, o)
	}
	return out
}

func envOrDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func origin(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	scheme := u.Scheme
	if scheme == "" {
		scheme = "https"
	}
	return scheme + "://" + stripPort(u.Host)
}

func hostFromPublicSiteURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	return stripPort(u.Host)
}

func stripPort(host string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}
