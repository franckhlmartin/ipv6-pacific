package ipv4outage

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestInOutageWindow(t *testing.T) {
	on := time.Date(2026, 6, 6, 12, 0, 0, 0, time.UTC)
	off := time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC)
	if !InOutageWindow(on) {
		t.Fatal("day 6 should be in window")
	}
	if InOutageWindow(off) {
		t.Fatal("day 7 should be out of window")
	}
}

func TestInPreOutageWindow(t *testing.T) {
	d4 := time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC)
	d6 := time.Date(2026, 6, 6, 0, 0, 0, 0, time.UTC)
	if !InPreOutageWindow(d4) {
		t.Fatal("day 4 pre-outage")
	}
	if InPreOutageWindow(d6) {
		t.Fatal("day 6 not pre-outage")
	}
	if DaysUntilOutage(d4) != 2 {
		t.Fatalf("days until=%d want 2", DaysUntilOutage(d4))
	}
}

func TestUnavailableUntil(t *testing.T) {
	t6 := time.Date(2026, 6, 6, 15, 0, 0, 0, time.UTC)
	u := UnavailableUntil(t6)
	want := time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC)
	if !u.Equal(want) {
		t.Fatalf("until=%v want %v", u, want)
	}
}

func TestAppliesToHost(t *testing.T) {
	cfg := Config{OutageHost: "pacific.ipv6forum.com"}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "pacific.ipv6forum.com"
	if !AppliesToHost(req, cfg) {
		t.Fatal("main host")
	}
	req.Host = "ipv4.pacific.ipv6forum.com"
	if AppliesToHost(req, cfg) {
		t.Fatal("probe host excluded")
	}
}

func TestShouldBlock_ipv4OnDay6(t *testing.T) {
	cfg := Config{OutageHost: "pacific.ipv6forum.com"}
	now := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	req := httptest.NewRequest(http.MethodGet, "/country/FJ", nil)
	req.Host = "pacific.ipv6forum.com"
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	if !ShouldBlock(req, cfg, now) {
		t.Fatal("expected block")
	}
}

func TestShouldBlock_ipv6Passes(t *testing.T) {
	cfg := Config{OutageHost: "pacific.ipv6forum.com"}
	now := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "pacific.ipv6forum.com"
	req.Header.Set("X-Forwarded-For", "2001:db8::1")
	if ShouldBlock(req, cfg, now) {
		t.Fatal("ipv6 should pass")
	}
}

func TestShouldBlock_loopbackExempt(t *testing.T) {
	cfg := Config{OutageHost: "pacific.ipv6forum.com", Force: true}
	now := time.Now()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "pacific.ipv6forum.com"
	req.RemoteAddr = "127.0.0.1:8082"
	if ShouldBlock(req, cfg, now) {
		t.Fatal("loopback exempt")
	}
}

func TestShouldBlock_embedExempt(t *testing.T) {
	cfg := Config{OutageHost: "pacific.ipv6forum.com"}
	now := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	for _, path := range []string{
		"/embed/conn-status",
		"/embed/conn-status/details",
		"/embed/conn-status.js",
		"/static/css/conn-status-embed.css",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Host = "pacific.ipv6forum.com"
		req.Header.Set("X-Forwarded-For", "203.0.113.1")
		if ShouldBlock(req, cfg, now) {
			t.Fatalf("embed exempt %s", path)
		}
	}
}

func TestShouldBlock_healthzExempt(t *testing.T) {
	cfg := Config{OutageHost: "pacific.ipv6forum.com"}
	now := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	req := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	req.Host = "pacific.ipv6forum.com"
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	if ShouldBlock(req, cfg, now) {
		t.Fatal("dual-stack probe healthz exempt on main host")
	}
}

func TestShouldBlock_crawlerExempt(t *testing.T) {
	cfg := Config{OutageHost: "pacific.ipv6forum.com"}
	now := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	for _, path := range []string{"/robots.txt", "/sitemap.xml", "/og/map.png"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Host = "pacific.ipv6forum.com"
		req.Header.Set("X-Forwarded-For", "203.0.113.1")
		if ShouldBlock(req, cfg, now) {
			t.Fatalf("crawler exempt %s", path)
		}
	}
}

func TestShouldBlock_skipAndForce(t *testing.T) {
	now := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "pacific.ipv6forum.com"
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	cfg := Config{OutageHost: "pacific.ipv6forum.com", Skip: true, Force: true}
	if ShouldBlock(req, cfg, now) {
		t.Fatal("skip wins")
	}
	cfg = Config{OutageHost: "pacific.ipv6forum.com", Force: true}
	if !ShouldBlock(req, cfg, now) {
		t.Fatal("force enables off-schedule")
	}
}

func TestServe566_headersAndHTML(t *testing.T) {
	tmpl := template.Must(template.New("566.html").Parse(`<p>until {{.ResumePlain}}</p>`))
	now := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	until := UnavailableUntil(now)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()
		Serve566(rec, req, tmpl, until, "tok123", nil)
	if rec.Code != 566 {
		t.Fatalf("status=%d", rec.Code)
	}
	if rec.Header().Get("Retry-Over-IPv6") != "?1" {
		t.Fatal("missing Retry-Over-IPv6")
	}
	if !strings.Contains(rec.Header().Get("Retry-Over-IPv6-Token"), "tok123") {
		t.Fatal("missing token header")
	}
	if !strings.Contains(rec.Body.String(), "until") {
		t.Fatal("expected html body")
	}
}

func TestServe566_problemJSON(t *testing.T) {
	now := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	until := UnavailableUntil(now)
	req := httptest.NewRequest(http.MethodGet, "/api/index.json", nil)
	rec := httptest.NewRecorder()
	Serve566(rec, req, nil, until, "abc", nil)
	if rec.Code != 566 {
		t.Fatalf("status=%d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/problem+json") {
		t.Fatalf("content-type=%q", ct)
	}
	if !strings.Contains(rec.Body.String(), `"retryOverIPv6":true`) {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestMiddleware_blocksIPv4(t *testing.T) {
	tmpl := template.Must(template.New("566.html").Parse("blocked"))
	cfg := Config{OutageHost: "pacific.ipv6forum.com", Force: true}
	now := func() time.Time { return time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC) }
	var hit bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusOK)
	})
	h := Middleware(cfg, tmpl, nil, now, next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "pacific.ipv6forum.com"
	req.Header.Set("X-Forwarded-For", "198.51.100.10")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if hit {
		t.Fatal("next should not run")
	}
	if rec.Code != 566 {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestPrefersProblemJSON_accept(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/country/FJ", nil)
	req.Header.Set("Accept", "application/problem+json")
	if !PrefersProblemJSON(req) {
		t.Fatal("should prefer problem+json")
	}
}
