package ripestat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const allowedHost = "stat.ripe.net"

// Client fetches RIPEstat Data API endpoints with hostname allowlisting.
type Client struct {
	HTTP      *http.Client
	SourceApp string
	// MinInterval between consecutive requests (rate discipline).
	MinInterval time.Duration
	lastReq     time.Time
}

func NewClient(hc *http.Client, sourceApp string, minInterval time.Duration) *Client {
	if hc == nil {
		hc = http.DefaultClient
	}
	if sourceApp == "" {
		sourceApp = "pacific-ipv6-monitor"
	}
	if minInterval <= 0 {
		minInterval = 300 * time.Millisecond
	}
	return &Client{HTTP: hc, SourceApp: sourceApp, MinInterval: minInterval}
}

type apiWrap struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *Client) wait(ctx context.Context) error {
	if c.MinInterval <= 0 {
		return nil
	}
	since := time.Since(c.lastReq)
	if since < c.MinInterval {
		d := c.MinInterval - since
		t := time.NewTimer(d)
		defer t.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
		}
	}
	c.lastReq = time.Now()
	return nil
}

func (c *Client) getJSON(ctx context.Context, endpoint string, params url.Values) ([]byte, error) {
	if err := c.wait(ctx); err != nil {
		return nil, err
	}
	if params == nil {
		params = url.Values{}
	}
	if c.SourceApp != "" {
		params.Set("sourceapp", c.SourceApp)
	}
	rawURL := fmt.Sprintf("https://%s/data/%s/data.json?%s", allowedHost, endpoint, params.Encode())
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "https" || parsed.Hostname() != allowedHost {
		return nil, errors.New("ripestat: host not allowlisted")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
		return nil, fmt.Errorf("ripestat: http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ripestat: http %d", resp.StatusCode)
	}
	return body, nil
}

func parseData(body []byte) (json.RawMessage, error) {
	var wrap apiWrap
	if err := json.Unmarshal(body, &wrap); err != nil {
		return nil, err
	}
	if strings.EqualFold(wrap.Status, "error") {
		return nil, fmt.Errorf("ripestat: api error: %s", wrap.Message)
	}
	return wrap.Data, nil
}
