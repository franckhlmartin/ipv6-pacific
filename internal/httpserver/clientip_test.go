package httpserver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestClientIPFamily_ipv4RemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	req.RemoteAddr = "203.0.113.5:12345"
	ip, family := ClientIPFamily(req)
	if ip != "203.0.113.5" || family != "ipv4" {
		t.Fatalf("got ip=%q family=%q", ip, family)
	}
}

func TestClientIPFamily_ipv6RemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	req.RemoteAddr = "[2001:db8::1]:443"
	ip, family := ClientIPFamily(req)
	if ip != "2001:db8::1" || family != "ipv6" {
		t.Fatalf("got ip=%q family=%q", ip, family)
	}
}

func TestClientIPFamily_xForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	req.Header.Set("X-Forwarded-For", "198.51.100.2, 10.0.0.1")
	req.RemoteAddr = "10.0.0.1:8080"
	ip, family := ClientIPFamily(req)
	if ip != "198.51.100.2" || family != "ipv4" {
		t.Fatalf("got ip=%q family=%q", ip, family)
	}
}

func TestConnectSrcFromProbeEnv_includesDS(t *testing.T) {
	t.Setenv("PROBE_V4_URL", "https://ipv4.example.com/api/healthz")
	t.Setenv("PROBE_V6_URL", "https://ipv6.example.com/api/healthz")
	t.Setenv("PROBE_DS_URL", "https://pacific.example.com/api/healthz")
	got := ConnectSrcFromProbeEnv()
	want := map[string]bool{
		"https://ipv4.example.com":   true,
		"https://ipv6.example.com":   true,
		"https://pacific.example.com": true,
	}
	if len(got) != len(want) {
		t.Fatalf("got %v want 3 origins", got)
	}
	for _, token := range got {
		if !want[token] {
			t.Fatalf("unexpected token %q in %v", token, got)
		}
	}
}

func TestHealthzCORSAllowOrigin_reflectsMatchingOrigin(t *testing.T) {
	t.Setenv("HEALTHZ_CORS_ALLOW_ORIGIN", "https://pacific.example.com,https://127.0.0.1:8082")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	req.Header.Set("Origin", "https://127.0.0.1:8082")
	HealthzCORSAllowOrigin(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://127.0.0.1:8082" {
		t.Fatalf("got ACAO %q", got)
	}
}

func TestConnectSrcFromProbeEnv_defaults(t *testing.T) {
	for _, k := range []string{"PROBE_V4_URL", "PROBE_V6_URL", "PROBE_DS_URL", "PUBLIC_SITE_URL"} {
		_ = os.Unsetenv(k)
	}
	got := ConnectSrcFromProbeEnv()
	want := map[string]bool{
		"https://ipv4.pacific.ipv6forum.com": true,
		"https://ipv6.pacific.ipv6forum.com": true,
		"https://pacific.ipv6forum.com":      true,
	}
	if len(got) != len(want) {
		t.Fatalf("got %v want 3 origins", got)
	}
	for _, token := range got {
		if !want[token] {
			t.Fatalf("unexpected token %q in %v", token, got)
		}
	}
}
