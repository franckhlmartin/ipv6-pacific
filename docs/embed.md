# Embed widget

Third-party sites can embed the Pacific Islands IPv6 Monitor **connection-status** control (button + details dialog). Probe endpoints are configured on **pacific-web** only — embedders cannot override them.

## Quick start (iframe — recommended)

```html
<iframe
  src="https://pacific.ipv6forum.com/embed/conn-status"
  title="Your IPv6 connection status"
  width="185" height="48"
  style="border:0;overflow:hidden"
  loading="lazy"
></iframe>
```

Replace the origin with your public site URL (`PUBLIC_SITE_URL` in production).

When visitors click the status button, connection details open in a **popup window** (`/embed/conn-status/details`) — no iframe resizing is needed on the host page.

## Script embed (advanced)

```html
<div id="ipv6-conn-status"></div>
<link rel="stylesheet" href="https://pacific.ipv6forum.com/static/css/conn-status-embed.css">
<script async src="https://pacific.ipv6forum.com/embed/conn-status.js"></script>
```

Optional: `data-mount="#your-selector"` on the script tag (default `#ipv6-conn-status`).

Probe URLs are baked into `/embed/conn-status.js` at server startup from `PROBE_V4_URL`, `PROBE_V6_URL`, and `PROBE_DS_URL`.

The script embed opens an **in-page dialog** on your site (unlike the iframe, which uses a popup for details).

## Host page CSP

| Embed type | Required directives |
|------------|---------------------|
| **iframe** | `frame-src https://pacific.ipv6forum.com` (or your public origin) |
| **script** | `script-src` and `connect-src` to your public origin **and** probe hostnames (`ipv4.*`, `ipv6.*`, dual-stack host) |

## Production nginx

The main site uses `X-Frame-Options: DENY` globally in the example nginx config. **iframe embed requires** dedicated `location =` blocks for embed paths that **omit** `X-Frame-Options` (Go sets `Content-Security-Policy: frame-ancestors *` on `/embed/conn-status`).

See [security.md](security.md) for the full nginx snippet and verification `curl` commands.

## Probe vhosts (operators)

Embed detection uses cross-origin `fetch()` to:

| Env | Role |
|-----|------|
| `PROBE_V4_URL` | IPv4-only vhost `/api/healthz` |
| `PROBE_V6_URL` | IPv6-only vhost `/api/healthz` |
| `PROBE_DS_URL` | Dual-stack vhost (preferred stack) |

Keep **`HEALTHZ_CORS_ALLOW_ORIGIN`** unset (default **`*`**) or remove **`HEALTHZ_CORS_RESTRICT`**. Do **not** set **`HEALTHZ_CORS_RESTRICT=1`** if third-party sites should embed the widget — probes on **ipv4**, **ipv6**, and the dual-stack main host must all return **`Access-Control-Allow-Origin: *`** for arbitrary embedder origins.

## Monthly 6/6 IPv4 drill

On UTC day 6, IPv4 clients on the main dual-stack hostname receive HTTP **566** for most paths. These paths stay available for third-party embeds:

| Path | Purpose |
|------|---------|
| `/embed/conn-status` | iframe document |
| `/embed/conn-status/details` | popup details page (iframe embed) |
| `/embed/conn-status.js` | script-tag entrypoint |
| `/static/css/conn-status-embed.css` | script-tag stylesheet |

The iframe page inlines all CSS/JS (no `/static/js` follow-up requests). **`/embed`** (instructions landing) is **not** exempt — IPv4 visitors may see 566 there during the drill.

The **566 error page** itself includes an inlined connection-status button so IPv4 users can see **“IPv4 only”** during the drill.

## Privacy

The widget shows client addresses as seen by probe servers (same as the main site dialog). Cross-site embed intentionally allows any page that loads the widget to trigger probe fetches; probe URLs are server-controlled only.

## Public landing page

**`GET /embed`** on the main site includes live preview and copy-paste snippets.
