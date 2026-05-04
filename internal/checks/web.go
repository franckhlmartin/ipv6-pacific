package checks

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pacific-monitor/pacific-monitor/internal/model"
)

func checkWeb(ctx context.Context, apex string, cfg Config, preferredWebURL string) (model.ServiceColumn, string, error) {
	col := model.ServiceColumn{Location: "-", Display: "[0] -/-/- [-]"}

	finalHost, baseURL, err := discoverWeb(ctx, apex, cfg.HTTPTimeout, preferredWebURL)
	if err != nil || finalHost == "" {
		if err != nil {
			col.Display = fmt.Sprintf("[0] error: %v", err)
		}
		col.Class = model.DeployUnknown
		return col, "", err
	}

	col.Location = classifyLocation(finalHost, apex)

	resolver := net.Resolver{PreferGo: true}
	addrs, err := resolver.LookupIPAddr(ctx, finalHost)
	if err != nil {
		col.Display = fmt.Sprintf("[?] %v", err)
		col.Class = model.DeployUnknown
		return col, finalHost, err
	}

	var v4s, v6s []net.IP
	for _, a := range addrs {
		if a.IP.To4() != nil {
			v4s = append(v4s, a.IP)
		} else if a.IP.To16() != nil && a.IP.To4() == nil {
			v6s = append(v6s, a.IP)
		}
	}
	v4s = uniqueIPs(v4s)
	v6s = uniqueIPs(v6s)

	v4cfg := len(v4s)
	v6cfg := len(v6s)

	v4op := 0
	if v4cfg > 0 && httpHeadFamily(ctx, baseURL, cfg.HTTPTimeout, "tcp4") {
		v4op = v4cfg
	}
	v6op := 0
	if v6cfg > 0 && httpHeadFamily(ctx, baseURL, cfg.HTTPTimeout, "tcp6") {
		v6op = v6cfg
	}

	col.IPv4 = model.ServiceMetrics{Configured: v4cfg, Reachable: v4cfg, Operational: v4op}
	col.IPv6 = model.ServiceMetrics{Configured: v6cfg, Reachable: v6cfg, Operational: v6op}
	col.Display = fmt.Sprintf("[%d] %d/%d/%d [%s]", 1, v6cfg, v6cfg, v6op, col.Location)
	col.Class = classifyService(col.IPv4, col.IPv6)
	return col, finalHost, nil
}

// webDiscoverCandidates builds GET URLs: optional yaml web_url first, then apex and www.apex.
func webDiscoverCandidates(apex, preferredWebURL string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
			raw = "https://" + raw
		}
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" {
			return
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			u.Scheme = "https"
		}
		if u.Path == "" {
			u.Path = "/"
		}
		s := u.String()
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	add(preferredWebURL)
	add("https://" + apex + "/")
	add("https://www." + apex + "/")
	return out
}

func discoverWeb(ctx context.Context, apex string, timeout time.Duration, preferredWebURL string) (host string, baseURL string, err error) {
	candidates := webDiscoverCandidates(apex, preferredWebURL)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		},
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	var lastErr error
	for _, raw := range candidates {
		// Each candidate needs its own deadline. A single shared timeout across URLs meant the
		// first slow host (often bare apex) could burn the whole budget and the next URL failed
		// with "context deadline exceeded" even when it would respond quickly (e.g. www.usp.ac.fj).
		ctxCand, cancelCand := context.WithTimeout(ctx, timeout)
		req, err := http.NewRequestWithContext(ctxCand, http.MethodGet, raw, nil)
		if err != nil {
			cancelCand()
			lastErr = err
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			cancelCand()
			lastErr = err
			continue
		}
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64*1024))
		resp.Body.Close()
		cancelCand()
		u, _ := url.Parse(raw)
		host = u.Hostname()
		if resp.Request != nil && resp.Request.URL != nil {
			host = resp.Request.URL.Hostname()
		}
		host = strings.TrimSuffix(strings.ToLower(host), ".")
		baseURL = "https://" + host + "/"
		return host, baseURL, nil
	}
	return "", "", lastErr
}

func httpHeadFamily(ctx context.Context, rawURL string, timeout time.Duration, tcpNetwork string) bool {
	ctx2, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			d := net.Dialer{Timeout: timeout}
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			return d.DialContext(ctx, tcpNetwork, net.JoinHostPort(host, port))
		},
	}
	client := &http.Client{Transport: tr, Timeout: timeout, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return fmt.Errorf("redirect")
		}
		return nil
	}}
	req, err := http.NewRequestWithContext(ctx2, http.MethodGet, rawURL, nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64*1024))
	resp.Body.Close()
	return resp.StatusCode < 500
}
