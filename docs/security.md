# Security

Pacific Islands IPv6 Monitor is a **read-only public dashboard** fed by a **batch measurement daemon**. There are **no accounts or SQL database** in v1.

Cross-check ideas against [`bookerpal/docs/security.md`](/Users/franck/code/bookerpal/docs/security.md) where applicable (adapted for this threat model).

## Threat model

- **Untrusted HTTP input**: path segments (`/country/{iso2}`), optional query strings. No user-generated HTML stored.
- **Untrusted data**: JSON artifacts under `data/` are written only by the collector; integrity depends on host security.
- **Outbound**: collector fetches **APNIC Labs** JSON from an **allowlisted host** (`data1.labs.apnic.net`) over HTTPS with verification.

## Controls implemented

- **CSP and security headers** via `internal/httpserver` (see `cmd/web/main.go`): `Content-Security-Policy`, `X-Content-Type-Options`, `Referrer-Policy`, `Permissions-Policy`; **HSTS** when TLS is enabled on the listener.
- **ISO2 allowlist** for `/country/` and `/api/countries/` from `config/pacific_iso2.yaml`.
- **Rate limiting** on `/api/*` (excluding `/api/healthz`).
- **HTML templates** use `html/template` auto-escaping for dynamic text.
- **APNIC client**: hostname allowlist in `internal/apniclabs`.
- **`/.well-known/security.txt`** placeholder — replace contact email for production.

## Supply chain

CI runs `go vet ./...` and `go test ./...`. Periodically run **`govulncheck ./...`** with an up-to-date Go toolchain; advisories depend on the standard library version you compile with.

## nginx / Caddy (production)

Terminate TLS at the reverse proxy and forward `X-Forwarded-For`. Example header parity with BookerPal-style deployments:

```nginx
add_header X-Content-Type-Options nosniff always;
add_header Referrer-Policy "strict-origin-when-cross-origin" always;
# CSP may duplicate Go middleware — prefer one layer only.
```
