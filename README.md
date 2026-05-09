# Pacific Islands IPv6 / DNSSEC deployment monitor

Go services:

- **`cmd/collector`** — measures configured domains (NIST-style DNS / Mail / Web + simplified DNSSEC), ingests **APNIC Labs** `v6economy/{CC}.json`, writes `data/countries/{ISO2}.json` and `data/index.json`.
- **`cmd/web`** — serves the Afrinic-inspired UI, JSON API, Pacific EEZ overview (`static/img/EEZ_Oceania.svg`), and a sortable home economies table; the map and Deploy % / IPv6 pref. % cells use the same red→green ramp. The header includes a **your connection** control (border colors, optional dialog with addresses seen by the service). `index.json` includes `deployment_score_pct` per economy (mean RowScore / 4 × 100). HTML pages emit canonical / Open Graph / Twitter meta tags; **`GET /og/map.png`** renders a share-preview PNG (same ramp as the map when APNIC Labs `preferred_pc_raw` is in `index.json`). **`GET /robots.txt`** is served at the site root. Set **`PUBLIC_SITE_URL`** when TLS terminates in front of the app so canonical and social URLs use the public origin (see `.env.example`).

Quick start:

```bash
cp .env.example .env.local
./scripts/gen_dev_certs.sh               # self-signed TLS → certs/*.pem (gitignored)
./scripts/start_collector.sh -run-once              # all countries once; add -country=FJ for Fiji only
./scripts/start_server.sh                # HTTPS on LISTEN (default :8082)
```

Then open **`https://127.0.0.1:8082/`** (trust the dev certificate when prompted).

**Documentation:** start at **[docs/index.md](docs/index.md)** — layered index; open linked files only as needed (small context window).
