# Domain lists: methodology

Details for **`config/domains/{ISO2}.yaml`**. Prerequisites and config layout: **[config-and-domains.md](config-and-domains.md)**.

## 1. Preconditions

- Economy **ISO2** must exist in **`config/pacific_iso2.yaml`** or the collector skips that country file.
- Filename: **`config/domains/{ISO2}.yaml`** (uppercase ISO2).

## 2. What to include

Research using official/government sites, regulators, established carriers. Target **government**, **critical infrastructure**, and **telecommunications**.

| Category | Examples |
|----------|----------|
| Government | Portal, PM/president, ministries, parliament, justice, immigration, statistics |
| Regulators / ICT | Telecom regulator, ccTLD operator when government-run |
| State enterprises | Power, incumbent telco/post, state development bank |
| Telecom | Mobile/fixed/ISP brands serving residents |
| Tourism | Official national tourism promotion |
| Education | Flagship **`*.ac.{cc}`** when relevant |
| Banking | Central bank; optional major retail bank (**`sector: Government`** often used for systemic institutions) |

**Regional orgs:** only under HQ economy; **`regional: true`**, **`sector: Regional`** — see [config-and-domains.md](config-and-domains.md).

Prefer **ccTLD** / **`gov.{cc}`** when canonical; non-ccTLD allowed if clearly official (e.g. tourism `.com`). Skip speculative domains.

## 3. One row = one monitored hostname

Registrable **apex** to check (e.g. `itc.gov.fj`). Avoid duplicate apex rows for the same owner unless monitoring intent differs.

Optional **`web_url`** when HTTPS is not at apex/`www` (`internal/config/config.go` → `DomainEntry`).

## 4. YAML fields

| Field | Notes |
|-------|--------|
| `domain` | Required; lowercased on load |
| `organization` | Recommended |
| `sector` | e.g. Government, Telecommunications, Regional, Education, Energy |
| `regional` | `true` only for regional bodies hosted under this economy |
| `web_url` | Optional alternate HTTPS URL |

## 5. Ordering

Alphabetical by `domain` unless a file documents an exception (e.g. leading comment).

## 6. Validation

1. `go build ./...`
2. `config.LoadDomainsFile("config/domains/XX.yaml")` must succeed (`internal/config`).

## 7. HQ changes

Move regional entries between country files; update **`docs/config-and-domains.md`** HQ table.
