package siteurl

import (
	"net/http"
	"net/url"
	"os"
	"strings"
)

// PublicOrigin returns the absolute origin (scheme + host, no trailing slash) for building
// canonical URLs, og:url, and og:image. Uses PUBLIC_SITE_URL when set; otherwise derives
// from the request (TLS or X-Forwarded-Proto, Host).
func PublicOrigin(r *http.Request) string {
	if base := strings.TrimSpace(os.Getenv("PUBLIC_SITE_URL")); base != "" {
		u, err := url.Parse(base)
		if err == nil && u.Scheme != "" && u.Host != "" {
			return strings.TrimSuffix(u.Scheme+"://"+u.Host, "/")
		}
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	} else if strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	host := r.Host
	if host == "" {
		host = "localhost"
	}
	return scheme + "://" + host
}

// AbsoluteURL joins the public origin with an absolute path (must start with '/').
func AbsoluteURL(r *http.Request, path string) string {
	if path == "" {
		path = "/"
	}
	if path[0] != '/' {
		path = "/" + path
	}
	return PublicOrigin(r) + path
}
