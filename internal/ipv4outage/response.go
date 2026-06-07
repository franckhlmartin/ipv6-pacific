package ipv4outage

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

const statusIPv4Unavailable = 566

// ProblemDetails is RFC 9457-shaped JSON per draft-martin-retry-over-ipv6.
type ProblemDetails struct {
	Type                 string `json:"type"`
	Title                string `json:"title"`
	Status               int    `json:"status"`
	Detail               string `json:"detail"`
	RetryOverIPv6        bool   `json:"retryOverIPv6"`
	IPv4UnavailableUntil string `json:"ipv4UnavailableUntil"`
}

// Page566Data is passed to 566.html.
type Page566Data struct {
	ResumePlain       string
	AboutURL          string
	Nonce             string
	InlineCSS         template.CSS
	InlineJS          template.JS
	ConnStatusVariant string
	SiteURL           string
	OutageToken       string
}

// Page566Enricher adds conn-status bundle fields before rendering 566.html.
type Page566Enricher func(r *http.Request, data *Page566Data)

// ProblemDetails566 builds the machine-readable 566 body.
func ProblemDetails566(until time.Time) ProblemDetails {
	return ProblemDetails{
		Type:                 "about:blank",
		Title:                "IPv4 Unavailable",
		Status:               statusIPv4Unavailable,
		Detail:               fmt.Sprintf("IPv4 unavailable until %s.", until.UTC().Format(time.RFC3339)),
		RetryOverIPv6:        true,
		IPv4UnavailableUntil: until.UTC().Format(time.RFC3339),
	}
}

func set566Headers(w http.ResponseWriter, until time.Time, token string) {
	w.Header().Set("Retry-Over-IPv6", "?1")
	w.Header().Set("IPv4-Unavailable-Until", until.UTC().Format(http.TimeFormat))
	if token != "" {
		w.Header().Set("Retry-Over-IPv6-Token", `"`+token+`"`)
	}
	sec := int(time.Until(until).Seconds())
	if sec < 0 {
		sec = 0
	}
	w.Header().Set("Retry-After", fmt.Sprintf("%d", sec))
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

// Serve566 writes a draft-compliant 566 response.
func Serve566(w http.ResponseWriter, r *http.Request, tmpl *template.Template, until time.Time, token string, enrich Page566Enricher) {
	set566Headers(w, until, token)
	w.WriteHeader(statusIPv4Unavailable)

	if PrefersProblemJSON(r) {
		w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(ProblemDetails566(until))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if tmpl == nil {
		_, _ = w.Write([]byte(plainFallbackBody(until)))
		return
	}
	data := Page566Data{
		ResumePlain:       until.UTC().Format("2 January 2006, 00:00 UTC"),
		AboutURL:          "/about#ipv6-day-drill",
		ConnStatusVariant: "outage566",
		OutageToken:       token,
	}
	if enrich != nil {
		enrich(r, &data)
	}
	_ = tmpl.Execute(w, data)
}

func plainFallbackBody(until time.Time) string {
	return fmt.Sprintf(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"><title>Site not available on this connection</title></head><body>
<p>This site is not available on your current Internet connection.</p>
<p>The Internet is moving to a newer protocol generation called IPv6. This service is not reachable over the older generation (IPv4) on your network. You probably cannot fix this yourself.</p>
<p>Contact your Internet provider or your organization's IT help desk and say: "I cannot reach this site — it may require IPv6, but my system does not seem to work with IPv6."</p>
<p>If this is a planned outage, service over the older connection may resume after %s.</p>
</body></html>`, until.UTC().Format("2 January 2006, 00:00 UTC"))
}
