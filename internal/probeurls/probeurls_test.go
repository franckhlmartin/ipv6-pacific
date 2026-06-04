package probeurls_test

import (
	"os"
	"testing"

	"github.com/pacific-monitor/pacific-monitor/internal/probeurls"
)

func TestLoad_defaults(t *testing.T) {
	for _, k := range []string{"PROBE_V4_URL", "PROBE_V6_URL", "PROBE_DS_URL", "PUBLIC_SITE_URL"} {
		_ = os.Unsetenv(k)
	}
	cfg := probeurls.Load()
	if cfg.V4 != "https://ipv4.pacific.ipv6forum.com/api/healthz" {
		t.Fatalf("V4=%q", cfg.V4)
	}
	if cfg.V6 != "https://ipv6.pacific.ipv6forum.com/api/healthz" {
		t.Fatalf("V6=%q", cfg.V6)
	}
	if cfg.DS != "https://pacific.ipv6forum.com/api/healthz" {
		t.Fatalf("DS=%q", cfg.DS)
	}
}

func TestLoad_publicSiteHost(t *testing.T) {
	t.Setenv("PUBLIC_SITE_URL", "https://monitor.example.org:443/")
	_ = os.Unsetenv("PROBE_V4_URL")
	_ = os.Unsetenv("PROBE_V6_URL")
	_ = os.Unsetenv("PROBE_DS_URL")

	cfg := probeurls.Load()
	if cfg.V4 != "https://ipv4.monitor.example.org/api/healthz" {
		t.Fatalf("V4=%q", cfg.V4)
	}
	if cfg.DS != "https://monitor.example.org/api/healthz" {
		t.Fatalf("DS=%q", cfg.DS)
	}
}

func TestLoad_envOverrides(t *testing.T) {
	t.Setenv("PROBE_V4_URL", "https://ipv4.custom.test/api/healthz")
	t.Setenv("PROBE_V6_URL", "https://ipv6.custom.test/api/healthz")
	t.Setenv("PROBE_DS_URL", "https://custom.test/api/healthz")

	cfg := probeurls.Load()
	if cfg.V4 != "https://ipv4.custom.test/api/healthz" {
		t.Fatalf("V4=%q", cfg.V4)
	}
}

func TestOrigins(t *testing.T) {
	cfg := probeurls.Config{
		V4: "https://ipv4.example.com/api/healthz",
		V6: "https://ipv6.example.com/api/healthz",
		DS: "https://pacific.example.com/api/healthz",
	}
	got := probeurls.Origins(cfg)
	want := map[string]bool{
		"https://ipv4.example.com":    true,
		"https://ipv6.example.com":    true,
		"https://pacific.example.com": true,
	}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for _, o := range got {
		if !want[o] {
			t.Fatalf("unexpected %q in %v", o, got)
		}
	}
}
