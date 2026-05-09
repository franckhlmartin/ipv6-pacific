# Configuration and monitored domains

## Layout

| Path | Role |
|------|------|
| `config/pacific_iso2.yaml` | Economies on the map and in APNIC Labs ingestion (Oceania PICs; **AU** / **NZ** omitted per product plan). |
| `config/domains/{ISO2}.yaml` | Apex domains measured for that economy (collector → `data/countries/{ISO2}.json`). |

## Regional bodies (headquarters rule)

List each regional organization **once**, under its **headquarters** economy’s YAML (not under every member). Update this table when facts change.

| Domain | HQ economy (ISO2) |
|--------|-------------------|
| `ffa.int` | SB |
| `spc.int` | NC |
| `forumsec.org` | FJ |
| `sprep.org` | WS |

Use **`sector: Regional`** and **`regional: true`** on those entries.

## Adding or editing domain lists

Step-by-step methodology: **[domains-methodology.md](domains-methodology.md)**.
