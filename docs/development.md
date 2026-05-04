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

## Deploy

See `scripts/push_to_prod.sh`. Set `PROD_DEST` to your `user@host:/path`. Builds Linux `pacific-web` and `pacific-collector`, rsyncs code + config — **never** ships `.env`.

## TLS / dual-stack border

Optional env **`PROBE_V4_URL`** and **`PROBE_V6_URL`** must point to two distinct hostnames that resolve **A-only** and **AAAA-only** respectively, with TLS SAN coverage, for the blue “dual-stack” browser border. Without them, the UI falls back to **IPv4 vs IPv6 connection** coloring only.
