package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pacific-monitor/pacific-monitor/internal/probeurls"
)

type cspNonceKey struct{}

// CSPNonce returns the per-request nonce set by SecurityHeaders (empty if missing).
func CSPNonce(r *http.Request) string {
	v, _ := r.Context().Value(cspNonceKey{}).(string)
	return v
}

// HealthzCORSAllowOrigin sets Access-Control-Allow-Origin for GET /api/healthz probe fetches.
// HEALTHZ_CORS_ALLOW_ORIGIN may be *, one origin, or comma-separated origins. When the request
// sends Origin and it matches an entry in the list, that origin is echoed (required for CORS).
func HealthzCORSAllowOrigin(w http.ResponseWriter, r *http.Request) {
	cfg := strings.TrimSpace(os.Getenv("HEALTHZ_CORS_ALLOW_ORIGIN"))
	allow := "*"
	if cfg != "" {
		allow = cfg
	}
	if r != nil {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" && cfg != "" && cfg != "*" {
			for _, part := range strings.Split(cfg, ",") {
				if strings.TrimSpace(part) == origin {
					allow = origin
					break
				}
			}
		}
	}
	w.Header().Set("Access-Control-Allow-Origin", allow)
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Max-Age", "86400")
}

// ClientIPFamily returns the client address and inet family (ipv4 or ipv6) as seen by the server.
func ClientIPFamily(r *http.Request) (ip, family string) {
	ip = RemoteIP(r)
	family = "ipv4"
	if IsIPv6Client(r) {
		family = "ipv6"
	}
	return ip, family
}

// ConnectSrcFromProbeEnv returns extra CSP connect-src tokens (scheme://host) for probe URLs.
func ConnectSrcFromProbeEnv() []string {
	return probeurls.Origins(probeurls.Load())
}

// embedFrameablePath returns true for routes that may be embedded in third-party iframes.
func embedFrameablePath(path string) bool {
	return path == "/embed/conn-status"
}

// SecurityHeaders adds baseline headers (CSP is relaxed for Leaflet from self + map tiles).
// Inline scripts in HTML pages must use nonce="{{.Nonce}}" and receive CSPNonce from the request context.
// extraConnectSrc adds origins for cross-origin fetch (e.g. IPv4/IPv6 probe health URLs).
func SecurityHeaders(extraConnectSrc []string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonceBytes := make([]byte, 16)
		_, _ = rand.Read(nonceBytes)
		nonce := base64.StdEncoding.EncodeToString(nonceBytes)
		r = r.WithContext(context.WithValue(r.Context(), cspNonceKey{}, nonce))

		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), camera=(), microphone=()")
		// Map tiles: allow OSM, allow unpkg for Leaflet (see templates). Tighten to self-hosted vendor later.
		// script-src uses a per-request nonce so small inline bootstraps (probe URLs) work without 'unsafe-inline'.
		connectParts := []string{
			"'self'",
			"https://unpkg.com",
			"https://www.google-analytics.com",
			"https://*.google-analytics.com",
			"https://stats.g.doubleclick.net",
		}
		for _, o := range extraConnectSrc {
			o = strings.TrimSpace(o)
			if o != "" {
				connectParts = append(connectParts, o)
			}
		}
		cspParts := []string{
			"default-src 'self'",
			fmt.Sprintf("script-src 'self' https://unpkg.com https://www.googletagmanager.com 'nonce-%s'", nonce),
			"style-src 'self' 'unsafe-inline' https://unpkg.com",
			"img-src 'self' data: https://*.tile.openstreetmap.org https://www.google-analytics.com https://www.googletagmanager.com",
			fmt.Sprintf("connect-src %s", strings.Join(connectParts, " ")),
			"font-src 'self' https://unpkg.com",
			"object-src 'none'",
			"base-uri 'self'",
		}
		if embedFrameablePath(r.URL.Path) {
			cspParts = append(cspParts, "frame-ancestors *")
		} else {
			cspParts = append(cspParts, "frame-ancestors 'none'")
		}
		cspParts = append(cspParts, "upgrade-insecure-requests")
		csp := strings.Join(cspParts, "; ")
		w.Header().Set("Content-Security-Policy", csp)
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		h.ServeHTTP(w, r)
	})
}

// RateLimit is a simple token bucket per remote IP.
type RateLimit struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64
	capacity int
}

type bucket struct {
	tokens float64
	last   time.Time
}

func NewRateLimit(rps float64, burst int) *RateLimit {
	return &RateLimit{
		buckets:  make(map[string]*bucket),
		rate:     rps,
		capacity: burst,
	}
}

func (rl *RateLimit) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	b, ok := rl.buckets[ip]
	if !ok {
		b = &bucket{tokens: float64(rl.capacity) - 1, last: time.Now()}
		rl.buckets[ip] = b
		return true
	}
	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.last = now
	b.tokens = minFloat(b.tokens+elapsed*rl.rate, float64(rl.capacity))
	if b.tokens < 1 {
		return false
	}
	b.tokens -= 1
	return true
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// RemoteIP returns client IP from X-Forwarded-For or RemoteAddr.
func RemoteIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func IsIPv6Client(r *http.Request) bool {
	ip := net.ParseIP(RemoteIP(r))
	if ip == nil {
		return false
	}
	return ip.To4() == nil
}
