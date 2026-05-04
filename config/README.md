# Monitoring configuration

- **`pacific_iso2.yaml`** — Economies included on the map and in APNIC ingestion (Oceania PICs; **AU** and **NZ** are omitted from the default list as in the product plan).
- **`domains/{ISO2}.yaml`** — Domains to measure for that economy. Regional bodies are listed **once** under their **headquarters** economy (e.g. `ffa.int` → `SB`, `spc.int` → `NC`, `forumsec.org` → `FJ`). Update this README when HQs change.
