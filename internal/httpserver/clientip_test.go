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

func TestConnectSrcFromProbeEnv_empty(t *testing.T) {
	for _, k := range []string{"PROBE_V4_URL", "PROBE_V6_URL", "PROBE_DS_URL"} {
		_ = os.Unsetenv(k)
	}
	if len(ConnectSrcFromProbeEnv()) != 0 {
		t.Fatal("expected no connect-src tokens")
	}
}
