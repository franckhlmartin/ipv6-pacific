# Pacific Islands IPv6 / DNSSEC deployment monitor

Go services:

- **`cmd/collector`** — measures configured domains (NIST-style DNS / Mail / Web + simplified DNSSEC), ingests **APNIC Labs** `v6economy/{CC}.json`, writes `data/countries/{ISO2}.json` and `data/index.json`.
- **`cmd/web`** — serves the Afrinic-inspired UI, JSON API, Pacific EEZ overview (`static/img/EEZ_Oceania.svg`), and a sortable home economies table; the map and Deploy % / IPv6 pref. % cells use the same red→green ramp. `index.json` includes `deployment_score_pct` per economy (mean RowScore / 8 × 100).

Quick start:

```bash
cp .env.example .env.local
./scripts/gen_dev_certs.sh               # self-signed TLS → certs/*.pem (gitignored)
./scripts/start_collector.sh -run-once              # all countries once; add -country=FJ for Fiji only
./scripts/start_server.sh                # HTTPS on LISTEN (default :8082)
```

Then open **`https://127.0.0.1:8082/`** (trust the dev certificate when prompted).

Documentation: `docs/development.md`, `docs/security.md`, `docs/commit-workflow.md`, `config/README.md`.
