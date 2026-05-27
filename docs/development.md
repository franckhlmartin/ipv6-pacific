# Development

## Prerequisites

- Go 1.22+
- Network access for DNS/HTTP measurement during collector runs

## Local setup

```bash
cp .env.example .env.local
# Edit DATA_DIR if needed (default ./data)
```

### TLS for local web

```bash
./scripts/gen_dev_certs.sh
```

This writes **`certs/cert.pem`** and **`certs/key.pem`** (gitignored). Then:

### Run web UI

```bash
./scripts/start_server.sh
```

Open **`https://127.0.0.1:8082/`** (accept the browser warning for the self-signed cert). Without collector output, the map and index may be empty — run the collector (below).

### Run collector

**Foreground, one full pass then exit** (all economies that have `config/domains/{ISO}.yaml`):

```bash
./scripts/start_collector.sh -run-once
```

**Foreground, only Fiji once then exit**:

```bash
./scripts/start_collector.sh -run-once -country=FJ
```

**Daemon** — collects **one country immediately**, then waits `COLLECTOR_PER_COUNTRY_INTERVAL` (default **10m** in code; production often **`4h`** via env), then continues through the rest in a **shuffled** order (new order each restart):

```bash
./scripts/start_collector.sh
```

**Daemon, start the rotation with Fiji first** (remaining economies in a shuffled order):

```bash
./scripts/start_collector.sh -country=FJ
```

**Background** (same flags; logs to `nohup.out` unless you redirect):

```bash
nohup ./scripts/start_collector.sh -country=FJ >> collector.log 2>&1 &
```

Use two terminals for **web + collector** during integration testing.

### Hurricane Electric BGP + APNIC per-ASN table

Each economy pass downloads the public **Hurricane Electric** country networks HTML (`bgp.he.net/country/{ISO2}`), scrapes per-ASN **IPv6 preferred** from **APNIC Labs** (`stats.labs.apnic.net/ipv6/{ISO2}` — the `drawTable()` ASN list), merges both into **`bgp_he_net`** on `data/countries/{ISO2}.json`, and the country page renders the combined table above the domain results when rows exist. APNIC-only ASNs show **N/A** for HE route columns.

For a fast integration check use **Tokelau (`-country=TK`)**: two APNIC ASNs and three domains in config.

Set **`COLLECTOR_SKIP_HE_BGP=1`** (see `.env.example`) to skip outbound HE requests and leave the previous HE snapshot in JSON (ops kill switch). Per-ASN APNIC stats are skipped when `exclude_apnic` is set on an economy.

## SEO and Open Graph

HTML responses include `meta`/`link` tags for description, canonical URL, Open Graph, and Twitter Cards (`cmd/web/templates/partials/seo.html`).

Set **`PUBLIC_SITE_URL`** in `.env` to your public HTTPS origin when TLS terminates in front of Go (reverse proxy, CDN). If unset, canonical and social URLs derive from each request’s **`Host`** (and forwarded HTTPS hints).

**`GET /og/map.png`** renders the EEZ overview map as a **1200×630 PNG** using embedded **`EEZ_Oceania.svg`** and **`data/index.json`**. Coloring matches the homepage EEZ map: **`pct-color-ramp.js`** stops and interpolation in **`internal/ogmap/ramp.go`**, and **only APNIC Labs `preferred_pc_raw`** drives the percentage (same rule as **`map-home.js`** — no deployment-score substitute). Gray means missing Labs data, same as in the browser. Responses include **`ETag`** and **`Cache-Control: public, max-age=300`**. On failure, the handler still returns **HTTP 200** with a small fallback PNG so `og:image` stays valid for crawlers.

Rasterization is pure Go (**oksvg** + **rasterx**).

### Sitemap (Google / Bing)

**`GET /sitemap.xml`** returns a [sitemaps.org](https://www.sitemaps.org/protocol.html) **urlset** for indexable HTML pages: home (`/`), about (`/about`), and one URL per economy in `config/pacific_iso2.yaml` as `/country/{ISO2}`. `lastmod` for `/` comes from `data/index.json`’s `generated_at`; for country pages it uses the on-disk mtime of `data/countries/{ISO2}.json` when that file exists.

Implementation: **`serveSitemap`** in [`cmd/web/sitemap.go`](../cmd/web/sitemap.go), registered in [`cmd/web/main.go`](../cmd/web/main.go). **`GET /robots.txt`** serves the embedded rules from `cmd/web/static/robots.txt` and appends a fully qualified **`Sitemap:`** line built with the same origin logic as canonical URLs (`siteurl`), so crawlers discover `/sitemap.xml` without hard-coding the public hostname.

**When you add a new public HTML route** that should be crawled, update **`serveSitemap`** in the same change so the new path appears in the urlset. Do **not** list JSON APIs (`/api/…`), static assets, **`/og/map.png`**, or health-style endpoints — align with [`cmd/web/static/robots.txt`](../cmd/web/static/robots.txt) (`Disallow: /api/`).

For production, set **`PUBLIC_SITE_URL`** so every `<loc>` and the robots **`Sitemap:`** URL use your real HTTPS origin (same as canonical / Open Graph).

## Commit workflow

See **`docs/commit-workflow.md`** (check changes since last push, doc updates, commit/push).

## Deploy

### Build and rsync

See `scripts/push_to_prod.sh`. Set `PROD_DEST` to your `user@host:/path`. It builds Linux **`pacific-web`** and **`pacific-collector`**, rsyncs code + config — **never** ships `.env`.

On the server, create **`/opt/ipv6-pacific/.env`** from `.env.example` (set `DATA_DIR`, `LISTEN`, `TLS_CERT_FILE`, `TLS_KEY_FILE`, etc.). **`pacific-web` and `pacific-collector` load `.env` / `.env.local` from the directory containing the executable (after resolving symlinks), then from the process working directory** — whichever comes first populates variables, and `godotenv` does not overwrite names already set in the environment. Prefer **`WorkingDirectory=/opt/ipv6-pacific`** so relative paths in `.env` (e.g. `PROJECT_ROOT=.`, cert paths) stay correct. If **`Environment=` or `EnvironmentFile=` pre-defines a key** (even as empty), values from `.env` for that key are skipped; remove duplicate keys from the unit if probes or other vars look unset. Run the collector as a separate service or cron so `data/` stays populated. **`data/` must be readable by the web service user** (e.g. `franck`). If you sometimes run the collector as **root**, set **`COLLECTOR_DATA_USER=franck`** (and optionally **`COLLECTOR_DATA_GROUP`**) so each successful run **`chown`s `DATA_DIR`** after writing; otherwise use **`chown -R franck:franck data`** or run the collector as **`User=franck`**. Without that, the UI can show “No index yet” even when `index.json` exists.

### systemd service (web)

Install a unit such as **`/etc/systemd/system/ipv6-pacific-web.service`** (adjust `User`, `Group`, and paths if needed):

```ini
[Unit]
Description=Pacific IPv6 Monitor (web)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=franck
Group=franck
WorkingDirectory=/opt/ipv6-pacific
# Optional: systemd can load env instead of/in addition to .env — use one consistent approach
# EnvironmentFile=/opt/ipv6-pacific/.env
ExecStart=/opt/ipv6-pacific/pacific-web
Restart=on-failure
RestartSec=5
# Hardening (adjust if something breaks)
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

Then:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now ipv6-pacific-web
sudo systemctl status ipv6-pacific-web
```

Put a reverse proxy in front if you terminate public TLS elsewhere; see `docs/security.md` for an **nginx** example (`pacific.ipv6forum.com`) and **`X-Forwarded-For`**.

## TLS / dual-stack border

Three optional probe URLs (full `https://host/.../api/healthz` paths, TLS SAN coverage):

| Env | Hostname role | Purpose |
|-----|----------------|---------|
| **`PROBE_V4_URL`** | A-only (e.g. `ipv4.pacific.ipv6forum.com`) | Can the browser reach the service over IPv4? |
| **`PROBE_V6_URL`** | AAAA-only (e.g. `ipv6.pacific.ipv6forum.com`) | Can the browser reach the service over IPv6? |
| **`PROBE_DS_URL`** | Dual-stack (e.g. `pacific.ipv6forum.com`) | Which stack did the browser **prefer** for this site? |

**`PROBE_V4_URL`** and **`PROBE_V6_URL`** together enable the blue “dual-stack” browser border. Without both, the UI falls back to **IPv4 vs IPv6 connection** coloring via **`/api/client-ip-family`**. **`PROBE_DS_URL`** is independent: it fills the dialog **Preferred for this site** row when set.

At startup, **`pacific-web` logs** probe configuration. The HTML injects `window.__PROBE_V4__`, `window.__PROBE_V6__`, and `window.__PROBE_DS__` (inline bootstrap **before** `border.js`). Probe **`fetch()`** requests hit those hostnames’ access logs. **`Content-Security-Policy` `connect-src`** is extended automatically from all three `PROBE_*` origins.

**`GET /api/healthz`** responses are JSON: **`{"ok":true,"ip":"...","family":"ipv4"|"ipv6"}`** (client address and inet family as seen on that request, using **`RemoteIP`** / **`X-Forwarded-For`**). The border script uses **`ip`** from v4/v6 probes in the dialog IPv4/IPv6 rows and **`ip`** + **`family`** from the DS probe for **Preferred for this site**. Responses include **`Access-Control-Allow-Origin`** (default `*`, override with **`HEALTHZ_CORS_ALLOW_ORIGIN`**) so the main page can read the body cross-origin. If a **reverse proxy** answers `/api/healthz` without forwarding to `pacific-web`, proxy to the app (or mirror CORS + JSON shape) on **ipv4**, **ipv6**, and **dual-stack** vhosts.

**`GET /api/client-ip-family`** returns **`family`** and **`ip`** for the browser’s connection to the **page origin** when v4/v6 probes are not configured; that endpoint is **rate-limited** like other `/api/*` routes (see `docs/security.md`). The header shows **IPv4 only**, **IPv6 only**, or **Dual stack** (matching table legend wording) with optional details in a dialog.

For **privacy and trust** assumptions when showing addresses in the UI, see **`docs/security.md`** (Client IP in UI).

## DMARC and RPKI (collector v0.3+)

- **DMARC**: `_dmarc.{apex}` TXT per domain in `internal/checks/dmarc.go`; stored on `DomainResult.dmarc`; country table column uses 0–100% ramp (`internal/rampscore`).
- **RPKI**: RIPEstat `announced-prefixes` + `rpki-validation` per ASN after HE/APNIC merge (`internal/collector/rpki.go`); sampled prefix cap via `COLLECTOR_RPKI_MAX_PREFIXES_PER_ASN`. Row score / economy deployment score **unchanged** in v1.
- **Ops**: email **stat@ripe.net** to register `RIPESTAT_SOURCEAPP` before large `run-once` bursts.

## Adding a new test column (contract)

When adding a new checker or changing checker output, keep collection logic, user-facing legend text, and score semantics aligned.

Required structure:

- Implement the checker in `internal/checks` and keep compact cell output deterministic.
- Add checker-owned legend metadata in the same package so UI text lives with the test logic:
  - update `internal/checks/legend.go` aggregator
  - provide a per-check explanation function in the checker file (pattern used by DNS/Mail/Web/DNSSEC)
- Use shared status classes consistently (`ipv4_only`, `dual_stack`, `ipv6_only`, `unknown`) for color semantics in the web table.

Required outcome semantics:

- Keep compact output decodable: include what each count/triplet means (Configured / Reachable / Operational).
- If a test has partial assurance (like DNSSEC currently), include explicit wording that avoids over-claiming.
- Unknown/error states must remain safe defaults and must not be scored as healthy.

Scoring integration rules:

- If the new test affects score, update `internal/scoring/score.go`:
  - `RowScore(...)` composition
  - point mapping rules
  - `MaxRowScore` if row maximum changes
  - `EconomyDeploymentScorePct(...)` denominator assumptions
- Update score legend text in `internal/scoring/legend.go` so `/country/{ISO}` explains the new formula exactly.

PR validation checklist for new tests:

- Country page renders with no template errors on `https://127.0.0.1:8082/country/FJ`.
- End-of-page legend explains the new checker format and meaning.
- Status colors and points are still consistent with scoring implementation.
- Lints pass for touched files and no existing behavior regresses for DNS/Mail/Web/DNSSEC.
