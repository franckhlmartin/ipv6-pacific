package ipv4outage

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLog566_emitsJSON(t *testing.T) {
	orig := log.Writer()
	defer log.SetOutput(orig)
	var buf strings.Builder
	log.SetOutput(&buf)

	req := httptest.NewRequest(http.MethodGet, "/country/FJ", nil)
	req.Host = "pacific.ipv6forum.com"
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	req.Header.Set("User-Agent", "Mozilla/5.0 TestBrowser")
	Log566(req, "tok123")

	line := strings.TrimSpace(buf.String())
	idx := strings.Index(line, "ipv4_outage ")
	if idx < 0 {
		t.Fatalf("line=%q", line)
	}
	payload := strings.TrimPrefix(line[idx:], "ipv4_outage ")
	var rec logRecord
	if err := json.Unmarshal([]byte(payload), &rec); err != nil {
		t.Fatal(err)
	}
	if rec.Event != "566" || rec.Token != "tok123" || rec.Path != "/country/FJ" {
		t.Fatalf("rec=%+v", rec)
	}
	if rec.ClientIP != "203.0.113.1" || rec.ClientFamily != "ipv4" {
		t.Fatalf("client=%+v", rec)
	}
	if rec.UserAgent == "" || rec.Host != "pacific.ipv6forum.com" {
		t.Fatalf("ua/host=%+v", rec)
	}
}

func TestLogProbe_onlyDuringOutage(t *testing.T) {
	orig := log.Writer()
	defer log.SetOutput(orig)
	var buf strings.Builder
	log.SetOutput(&buf)

	cfg := Config{OutageHost: "pacific.ipv6forum.com"}
	off := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	on := time.Date(2026, 6, 6, 12, 0, 0, 0, time.UTC)

	req := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	req.Host = "pacific.ipv6forum.com"
	req.Header.Set("X-Forwarded-For", "2001:db8::1")

	LogProbe(req, cfg, off)
	if buf.Len() != 0 {
		t.Fatal("expected no log outside outage window")
	}

	LogProbe(req, cfg, on)
	if !strings.Contains(buf.String(), `"event":"probe"`) {
		t.Fatalf("line=%q", buf.String())
	}
}
