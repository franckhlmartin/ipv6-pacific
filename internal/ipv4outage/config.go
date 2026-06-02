package ipv4outage

import (
	"log"
	"net/url"
	"os"
	"strings"
)

const defaultOutageHost = "pacific.ipv6forum.com"

// Config holds IPv4 outage policy from the environment.
type Config struct {
	OutageHost string
	Skip       bool
	Force      bool
}

// LoadConfig reads IPV4_OUTAGE_* and PUBLIC_SITE_URL.
func LoadConfig() Config {
	cfg := Config{
		OutageHost: strings.TrimSpace(os.Getenv("IPV4_OUTAGE_HOST")),
		Skip:       strings.TrimSpace(os.Getenv("IPV4_OUTAGE_SKIP")) == "1",
		Force:      strings.TrimSpace(os.Getenv("IPV4_OUTAGE_FORCE")) == "1",
	}
	if cfg.OutageHost == "" {
		if h := hostFromPublicSiteURL(os.Getenv("PUBLIC_SITE_URL")); h != "" {
			cfg.OutageHost = h
		} else {
			cfg.OutageHost = defaultOutageHost
		}
	}
	return cfg
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

// WarnForceInProduction logs if IPV4_OUTAGE_FORCE is enabled.
func WarnForceInProduction(cfg Config) {
	if cfg.Force {
		log.Print("ipv4_outage: WARNING IPV4_OUTAGE_FORCE=1 — IPv4 clients may receive 566 on the main site outside the monthly schedule")
	}
}
