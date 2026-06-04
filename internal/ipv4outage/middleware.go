package ipv4outage

import (
	"html/template"
	"net/http"
	"time"
)

// Middleware enforces monthly IPv4 outage policy before the application mux.
func Middleware(cfg Config, tmpl566 *template.Template, enrich Page566Enricher, now func() time.Time, next http.Handler) http.Handler {
	if now == nil {
		now = time.Now
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := now()

		if OutageActive(cfg, t) && AppliesToHost(r, cfg) {
			if ok, tok := ParseRecoveryHeader(r); ok {
				LogRecovery(r, tok)
			}
		}

		if ShouldBlock(r, cfg, t) {
			token, err := NewToken()
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			until := UnavailableUntil(t)
			Log566(r, token)
			Serve566(w, r, tmpl566, until, token, enrich)
			return
		}

		next.ServeHTTP(w, r)
	})
}
