package ipv4outage

import (
	"log"
	"net/http"

	"github.com/pacific-monitor/pacific-monitor/internal/httpserver"
)

// Log566 records a 566 response for operations metrics.
func Log566(r *http.Request, token string) {
	log.Printf("ipv4_outage event=566 token=%s path=%s client_ip=%s host=%s",
		token, safePath(r), httpserver.RemoteIP(r), requestHost(r))
}

// LogRecovery records Retry-Over-IPv6-Recovery telemetry.
func LogRecovery(r *http.Request, token string) {
	log.Printf("ipv4_outage event=recovery token=%s path=%s client_ip=%s host=%s",
		token, safePath(r), httpserver.RemoteIP(r), requestHost(r))
}

func safePath(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	return r.URL.Path
}
