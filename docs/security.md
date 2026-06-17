# Security

Pacific Islands IPv6 Monitor is a **read-only public dashboard** fed by a **batch measurement daemon**. There are **no accounts or SQL database** in v1.

Cross-check ideas against [`bookerpal/docs/security.md`](/Users/franck/code/bookerpal/docs/security.md) where applicable (adapted for this threat model).

## Threat model

- **Untrusted HTTP input**: path segments (`/country/{iso2}`), optional query strings. No user-generated HTML stored.
- **Untrusted data**: JSON artifacts under `data/` are written only by the collector; integrity depends on host security.
- **Outbound**: collector fetches **APNIC Labs** JSON from **`data1.labs.apnic.net`**, scrapes per-ASN stats from **`stats.labs.apnic.net`**, **Hurricane Electric** from **`bgp.he.net`**, and **RIPEstat** from **`stat.ripe.net`** (RPKI sampling) — each over HTTPS with hostname allowlists in the respective clients. DMARC uses public DNS (resolver in `internal/checks`).

## Controls implemented

- **CSP and security headers** via `internal/httpserver` (see `cmd/web/main.go`): `Content-Security-Policy`, `X-Content-Type-Options`, `Referrer-Policy`, `Permissions-Policy`; **HSTS** when TLS is enabled on the listener.
- **ISO2 allowlist** for `/country/` and `/api/countries/` from `config/pacific_iso2.yaml`.
- **Rate limiting** on `/api/*` (excluding `/api/healthz`).
- **HTML templates** use `html/template` auto-escaping for dynamic text.
- **APNIC clients**: hostname allowlists in `internal/apniclabs` (`data1.labs.apnic.net`) and `internal/apnicstats` (`stats.labs.apnic.net`).
- **Hurricane Electric client**: hostname allowlist in `internal/bgphe` (`bgp.he.net`).
- **RIPEstat client**: hostname allowlist in `internal/ripestat` (`stat.ripe.net`); sequential requests with `sourceapp`; optional `COLLECTOR_SKIP_RPKI=1`. If daily volume regularly exceeds ~1,000 requests, register with **stat@ripe.net** (see `.env.example`).
- **`/.well-known/security.txt`** placeholder — replace contact email for production.

## Client IP in UI

The site may show **the visitor’s address as seen by the server** (header control → dialog: **Preferred for this site**, IPv4, and/or IPv6 rows; **Not available** when a probe fails or returns no `ip`). Addresses come from **`internal/httpserver.RemoteIP`**, which prefers **`X-Forwarded-For`** (first hop) when set. **Misconfigured proxies** can make the value wrong (edge IP, not the browser). Align with the **nginx** example in this doc (`proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for`).

**`GET /api/healthz`** embeds **`ip`** and **`family`** (`ipv4` or `ipv6`) in JSON for cross-origin reads used by v4, v6, and dual-stack (`PROBE_DS_URL`) probes, and for same-origin fallback when cross-origin probes fail. CORS defaults to **`Access-Control-Allow-Origin: *`**. An optional **`HEALTHZ_CORS_ALLOW_ORIGIN`** allowlist echoes a matching **`Origin`**; other origins still receive **`*`** unless **`HEALTHZ_CORS_RESTRICT=1`** (that mode blocks third-party embed). Do not set a comma-separated allowlist without understanding that embed requires open CORS on **all three** probe hostnames (including the dual-stack main host for **`PROBE_DS_URL`**). **`/api/healthz`** is excluded from rate limiting (unlike other `/api/*` routes).

## Embed widget

Third-party sites can embed the connection-status control via **`GET /embed/conn-status`** (iframe) or **`GET /embed/conn-status.js`** (script tag). Probe URLs are baked into the script at server startup — embedders cannot retarget probes.

- Cross-site embed requires **`Access-Control-Allow-Origin: *`** on **`GET /api/healthz`** for **ipv4**, **ipv6**, and **dual-stack** probe hostnames (default). Setting **`HEALTHZ_CORS_RESTRICT=1`** disables the fallback and breaks embed on arbitrary sites.
- During the **6/6 IPv4 drill**, embed asset paths are exempt from 566 (see [embed.md](embed.md)); the **566 HTML page** includes an inlined widget (not a public route).
- Full operator guide: [embed.md](embed.md).

## Monthly IPv4 outage (566)

The **6/6 IPv4 drill** (`internal/ipv4outage`) classifies clients using the same **`X-Forwarded-For`** (first hop) / **`RemoteIP`** rules as the connection UI. **Only trust this policy when nginx is the sole component setting `X-Forwarded-For`** toward `pacific-web` — do not forward client-supplied XFF from the Internet.

566 responses include **`Cache-Control: private, no-store`**, **`X-Content-Type-Options: nosniff`**, and the draft retry headers. **`Retry-Over-IPv6-Token`** values are logged for metrics, not used for authentication. **`566.html` is not registered as a public route** (rendered only by middleware).

See **`docs/development.md`** (Monthly 6/6 IPv4 outage) for ops variables and testing.

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

    # Security headers (X-Frame-Options only on location / — see embed locations below)
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

    # --- Embed widget (third-party iframe) ---
    # Go sets CSP frame-ancestors * on /embed/conn-status; do NOT send X-Frame-Options here.
    location = /embed/conn-status {
        proxy_pass https://localhost:8082;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        add_header X-Content-Type-Options nosniff;
        add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    }

    location = /embed/conn-status/details {
        proxy_pass https://localhost:8082;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        add_header X-Content-Type-Options nosniff;
        add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    }

    location = /embed/conn-status.js {
        proxy_pass https://localhost:8082;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        add_header X-Content-Type-Options nosniff;
    }

    location = /static/css/conn-status-embed.css {
        proxy_pass https://localhost:8082;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        add_header X-Content-Type-Options nosniff;
    }

    location / {
        add_header X-Frame-Options DENY always;
        add_header X-Content-Type-Options nosniff;
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

Verify embed framing:

```bash
curl -sI https://pacific.ipv6forum.com/embed/conn-status | grep -i frame
curl -sI https://pacific.ipv6forum.com/ | grep -i frame
```

The iframe path must **not** return `X-Frame-Options: DENY`; the home page must still deny framing.

### Combined site + probe vhosts (single server block)

When **`pacific.ipv6forum.com`**, **`ipv4.pacific.ipv6forum.com`**, and **`ipv6.pacific.ipv6forum.com`** share one **`server { }`** block, apply the same rules:

- **Do not** set **`X-Frame-Options DENY`** at **`server`** scope — it applies to embed paths too. Set it only on **`location /`** (as above).
- Keep dedicated **`location =`** blocks for embed paths on the main hostname (paths are served from the same upstream).
- Probe hostnames only need **`location /`** → upstream; CORS is set by **`pacific-web`**.

Verify CORS for third-party embed (expect **`access-control-allow-origin: *`** unless **`HEALTHZ_CORS_RESTRICT=1`**):

```bash
curl -sI -H "Origin: https://www.example.com" \
  "https://ipv4.pacific.ipv6forum.com/api/healthz" | grep -i access-control-allow-origin
curl -sI -H "Origin: https://www.example.com" \
  "https://ipv6.pacific.ipv6forum.com/api/healthz" | grep -i access-control-allow-origin
curl -sI -H "Origin: https://www.example.com" \
  "https://pacific.ipv6forum.com/api/healthz" | grep -i access-control-allow-origin
```
