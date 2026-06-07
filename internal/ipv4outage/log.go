package ipv4outage

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/pacific-monitor/pacific-monitor/internal/httpserver"
)

const maxLoggedUA = 120

type logRecord struct {
	Event        string `json:"event"`
	Token        string `json:"token,omitempty"`
	Path         string `json:"path,omitempty"`
	ClientIP     string `json:"client_ip,omitempty"`
	ClientFamily string `json:"client_family,omitempty"`
	UserAgent    string `json:"user_agent,omitempty"`
	Host         string `json:"host,omitempty"`
	Family       string `json:"family,omitempty"`
	Referer      string `json:"referer,omitempty"`
}

func writeLog(rec logRecord) {
	b, err := json.Marshal(rec)
	if err != nil {
		return
	}
	log.Printf("ipv4_outage %s", string(b))
}

func truncateUA(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLoggedUA {
		return s
	}
	return s[:maxLoggedUA]
}

func clientFields(r *http.Request) (ip, family, ua string) {
	if r == nil {
		return "", "ipv4", ""
	}
	ip, family = httpserver.ClientIPFamily(r)
	return ip, family, truncateUA(r.UserAgent())
}

// Log566 records a 566 response for operations metrics.
func Log566(r *http.Request, token string) {
	ip, family, ua := clientFields(r)
	writeLog(logRecord{
		Event:        "566",
		Token:        token,
		Path:         safePath(r),
		ClientIP:     ip,
		ClientFamily: family,
		UserAgent:    ua,
		Host:         requestHost(r),
	})
}

// LogRecovery records Retry-Over-IPv6-Recovery telemetry.
func LogRecovery(r *http.Request, token string) {
	ip, family, ua := clientFields(r)
	writeLog(logRecord{
		Event:        "recovery",
		Token:        token,
		Path:         safePath(r),
		ClientIP:     ip,
		ClientFamily: family,
		UserAgent:    ua,
		Host:         requestHost(r),
	})
}

// LogProbe records /api/healthz during an active outage (conn-status probes).
func LogProbe(r *http.Request, cfg Config, now time.Time) {
	if r == nil || !OutageActive(cfg, now) || !AppliesToHost(r, cfg) {
		return
	}
	if safePath(r) != "/api/healthz" {
		return
	}
	ip, family, ua := clientFields(r)
	writeLog(logRecord{
		Event:        "probe",
		Path:         safePath(r),
		ClientIP:     ip,
		ClientFamily: family,
		Family:       family,
		UserAgent:    ua,
		Host:         requestHost(r),
		Referer:      strings.TrimSpace(r.Referer()),
	})
}

func safePath(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	return r.URL.Path
}
