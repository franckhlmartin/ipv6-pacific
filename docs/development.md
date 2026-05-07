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

**Daemon** — collects **one country immediately**, then waits `COLLECTOR_PER_COUNTRY_INTERVAL`, then continues round-robin:

```bash
./scripts/start_collector.sh
```

**Daemon, start the rotation with Fiji first** (next tick continues after FJ in `pacific_iso2.yaml` order):

```bash
./scripts/start_collector.sh -country=FJ
```

**Background** (same flags; logs to `nohup.out` unless you redirect):

```bash
nohup ./scripts/start_collector.sh -country=FJ >> collector.log 2>&1 &
```

Use two terminals for **web + collector** during integration testing.

## Commit workflow

See **`docs/commit-workflow.md`** (check changes since last push, doc updates, commit/push).

## Deploy

### Build and rsync

See `scripts/push_to_prod.sh`. Set `PROD_DEST` to your `user@host:/path`. It builds Linux **`pacific-web`** and **`pacific-collector`**, rsyncs code + config — **never** ships `.env`.

On the server, create **`/opt/ipv6-pacific/.env`** from `.env.example` (set `DATA_DIR`, `LISTEN`, `TLS_CERT_FILE`, `TLS_KEY_FILE`, etc.). The web binary loads `.env` / `.env.local` from **`WorkingDirectory`**. Run the collector as a separate service or cron so `data/` stays populated. **`data/` must be readable by the web service user** (e.g. `franck`). If you sometimes run the collector as **root**, set **`COLLECTOR_DATA_USER=franck`** (and optionally **`COLLECTOR_DATA_GROUP`**) so each successful run **`chown`s `DATA_DIR`** after writing; otherwise use **`chown -R franck:franck data`** or run the collector as **`User=franck`**. Without that, the UI can show “No index yet” even when `index.json` exists.

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

Optional env **`PROBE_V4_URL`** and **`PROBE_V6_URL`** must point to two distinct hostnames that resolve **A-only** and **AAAA-only** respectively, with TLS SAN coverage, for the blue “dual-stack” browser border. Without them, the UI falls back to **IPv4 vs IPv6 connection** coloring only.

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
