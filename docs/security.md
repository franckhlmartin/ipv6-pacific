# Security

Pacific Islands IPv6 Monitor is a **read-only public dashboard** fed by a **batch measurement daemon**. There are **no accounts or SQL database** in v1.

Cross-check ideas against [`bookerpal/docs/security.md`](/Users/franck/code/bookerpal/docs/security.md) where applicable (adapted for this threat model).

## Threat model

- **Untrusted HTTP input**: path segments (`/country/{iso2}`), optional query strings. No user-generated HTML stored.
- **Untrusted data**: JSON artifacts under `data/` are written only by the collector; integrity depends on host security.
- **Outbound**: collector fetches **APNIC Labs** JSON from **`data1.labs.apnic.net`**, scrapes per-ASN stats from **`stats.labs.apnic.net`**, and **Hurricane Electric** from **`bgp.he.net`** — each over HTTPS with hostname allowlists in the respective clients.

## Controls implemented

- **CSP and security headers** via `internal/httpserver` (see `cmd/web/main.go`): `Content-Security-Policy`, `X-Content-Type-Options`, `Referrer-Policy`, `Permissions-Policy`; **HSTS** when TLS is enabled on the listener.
- **ISO2 allowlist** for `/country/` and `/api/countries/` from `config/pacific_iso2.yaml`.
- **Rate limiting** on `/api/*` (excluding `/api/healthz`).
- **HTML templates** use `html/template` auto-escaping for dynamic text.
- **APNIC clients**: hostname allowlists in `internal/apniclabs` (`data1.labs.apnic.net`) and `internal/apnicstats` (`stats.labs.apnic.net`).
- **Hurricane Electric client**: hostname allowlist in `internal/bgphe` (`bgp.he.net`).
- **`/.well-known/security.txt`** placeholder — replace contact email for production.

## Client IP in UI

The site may show **the visitor’s address as seen by the server** (header control → dialog: IPv4 and/or IPv6 rows; **Not available** when a probe fails or returns no `ip`). Addresses come from **`internal/httpserver.RemoteIP`**, which prefers **`X-Forwarded-For`** (first hop) when set. **Misconfigured proxies** can make the value wrong (edge IP, not the browser). Align with the **nginx** example in this doc (`proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for`).

**`GET /api/healthz`** embeds **`ip`** in JSON for cross-origin reads used by the dual-stack probe; CORS defaults to **`Access-Control-Allow-Origin: *`** unless **`HEALTHZ_CORS_ALLOW_ORIGIN`** is set. In production, **restricting** that variable to the **canonical site origin** reduces cross-site exfiltration of the response body to arbitrary malicious origins (see `docs/development.md`).

**`GET /api/client-ip-family`** exposes **`ip`** and **`family`** and is **subject to rate limiting** (unlike `/api/healthz`). No extra logging is added for modal-only use beyond normal request logs.

## Supply chain

CI runs `go vet ./...` and `go test ./...`. Periodically run **`govulncheck ./...`** with an up-to-date Go toolchain; advisories depend on the standard library version you compile with.

## nginx / Caddy (production)

Terminate TLS at the reverse proxy and forward **`X-Forwarded-For`** (and related headers) so rate limiting and client IP detection in `internal/httpserver` see the real client. The Go app also emits security headers; **avoid duplicating the same header in nginx and in Go** — pick one layer for CSP in particular.

### Example production site (`pacific.ipv6forum.com`)

Deployed as **`/etc/nginx/conf.d/ipv6-pacific.conf`** on `bookerpal-main`: HTTP redirects to HTTPS; HTTPS proxies to **`pacific-web`** on **`https://localhost:8082`** (the binary uses TLS; nginx must trust or verify that upstream as configured). Public TLS certificates live under **`/opt/ipv6-pacific/certs/`** (`fullchain.pem`, `key.pem`). Adjust paths, hostnames, and cipher lists for your environment.

```nginx
# Redirect HTTP pacific.ipv6forum.com to HTTPS
server {
    listen 80;
    listen [::]:80;
    server_name pacific.ipv6forum.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl;
    listen [::]:443 ssl;
    server_name pacific.ipv6forum.com;
    http2 on;

    # SSL Configuration
    ssl_certificate /opt/ipv6-pacific/certs/fullchain.pem;
    ssl_certificate_key /opt/ipv6-pacific/certs/key.pem;

    # SSL Security
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;

    # Security headers
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    # Content-Security-Policy (CSP)
    #add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' https:; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://cdnjs.cloudflare.com; img-src 'self' data: https:; font-src 'self' https://fonts.gstatic.com https://cdnjs.cloudflare.com; connect-src 'self'; object-src 'none'; frame-ancestors 'none'; upgrade-insecure-requests; block-all-mixed-content" always;

    # Referrer-Policy
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    # Permissions-Policy
    add_header Permissions-Policy "geolocation=(self \"https://pacific.ipv6forum.com\"), payment=(self \"https://pacific.ipv6forum.com\"), microphone=(), camera=()" always;

    # File upload size
    client_max_body_size 50M;

    #enabling compression to speed up load
    brotli on;
    brotli_types text/plain text/css application/json application/javascript text/xml application/xml application/xml+rss text/javascript;
    brotli_comp_level 6;
    brotli_min_length 256;

    location / {
        proxy_pass https://localhost:8082;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket support
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;

        # Keep typical proxied responses (e.g. cover images ~200KB+) in RAM so nginx
        # does not buffer to disk (/var/lib/nginx/tmp/proxy/...) and log a warning.
        # Defaults are often ~32–64KB total; raise if you serve larger assets via the app.
        proxy_buffer_size          16k;
        proxy_buffers              32 16k;   # 512KB body buffer pool
        proxy_busy_buffers_size    256k;     # must be < sum(proxy_buffers) minus one buffer
    }

}
```

Reload nginx after changes: `sudo nginx -t && sudo systemctl reload nginx`.
