#!/usr/bin/env python3
"""
Post-drill report for the monthly IPv4 outage (HTTP 566).

Reads journald ipv4_outage lines and nginx access logs, prints counts and
percentages by connection stack and User-Agent family.

Usage (on production, from repo root):
  ./scripts/ipv4_outage_report.sh --date 2026-06-06
  ./scripts/ipv4_outage_report.sh --date 2026-06-06 -o /tmp/drill.txt
  ./scripts/ipv4_outage_report.sh --journal-file fixtures/journal.txt --nginx-log fixtures/nginx.log --date 2026-06-06
"""

from __future__ import annotations

import argparse
import json
import re
import socket
import subprocess
import sys
import time
import urllib.error
import urllib.request
from collections import defaultdict
from dataclasses import dataclass, field
from datetime import date, datetime, timedelta
from pathlib import Path
from typing import DefaultDict, Iterable, Iterator, Optional, Set, Tuple

DEFAULT_SERVICE = "ipv6-pacific-web"
DEFAULT_MAIN_HOST = "pacific.ipv6forum.com"
NGINX_DIR = Path("/var/log/nginx")

PACIFIC_ISO2 = frozenset(
    {
        "AS",
        "AU",
        "CK",
        "FJ",
        "FM",
        "GU",
        "KI",
        "MH",
        "MP",
        "NC",
        "NR",
        "NU",
        "NZ",
        "PF",
        "PG",
        "PW",
        "SB",
        "TK",
        "TO",
        "TV",
        "US",
        "VU",
        "WS",
    }
)

EXEMPT_PATHS = frozenset(
    {
        "/robots.txt",
        "/sitemap.xml",
        "/og/map.png",
        "/embed/conn-status",
        "/embed/conn-status/details",
        "/embed/conn-status.js",
        "/static/css/conn-status-embed.css",
        "/api/healthz",
    }
)

JOURNAL_RE = re.compile(
    r"ipv4_outage event=(?P<event>566|recovery)\s+"
    r"token=(?P<token>\S+)\s+"
    r"path=(?P<path>\S+)\s+"
    r"client_ip=(?P<client_ip>\S+)\s+"
    r"host=(?P<host>\S+)"
)

# JSON log lines (future / mixed deployments)
JOURNAL_JSON_RE = re.compile(r"ipv4_outage\s+(\{.*\})")

NGINX_STATUS_RE = re.compile(r'"[A-Z]+ [^"]*"\s+(\d+)\s+')
NGINX_DATE_RE = re.compile(r"\[(\d{2}/\w{3}/\d{4}):")
UA_VHOST_RE = re.compile(r' "-" "([^"]*)" "-" "([^"]*)"\s*$')
NGINX_PATH_RE = re.compile(r"^[A-Z]+\s+(\S+)")


@dataclass
class JournalEvent:
    event: str
    token: str
    path: str
    client_ip: str
    host: str
    hour: Optional[int] = None


@dataclass
class NginxRow:
    remote_addr: str
    status: int
    vhost: str
    user_agent: str
    path: str
    stack: str  # ipv4 | ipv6


@dataclass
class BucketStats:
    requests: int = 0
    blocked: int = 0
    ips: Set[str] = field(default_factory=set)


@dataclass
class FamilyStats:
    total: int = 0
    ipv4_requests: int = 0
    ipv6_requests: int = 0
    blocked: int = 0
    ipv4_ips: Set[str] = field(default_factory=set)
    ipv6_ips: Set[str] = field(default_factory=set)


def normalize_path(path: str) -> str:
    p = (path or "/").split("?")[0].split("#")[0]
    if not p.startswith("/"):
        p = "/" + p
    return p.rstrip("/") or "/"


def is_exempt_path(path: str) -> bool:
    """Paths that stay reachable on IPv4 during the drill (see internal/ipv4outage/block.go)."""
    p = normalize_path(path)
    if p in EXEMPT_PATHS:
        return True
    # Trailing-slash / subpath variants
    for ep in EXEMPT_PATHS:
        ep_norm = normalize_path(ep)
        if p == ep_norm or p.startswith(ep_norm + "/"):
            return True
    return False


def count_with_pct(n: int, total: int) -> str:
    if total == 0:
        return f"{n} (—)"
    return f"{n} ({pct(n, total)})"


def pct(num: int, denom: int) -> str:
    if denom == 0:
        return "—"
    return f"{100.0 * num / denom:.1f}%"


def connection_stack(remote_addr: str) -> str:
    if ":" in remote_addr:
        return "ipv6"
    return "ipv4"


def classify_ua(ua: str) -> Tuple[str, str]:
    """Return (family, form_factor). Order matters."""
    u = (ua or "").strip()
    if not u or u == "-":
        return "Empty/unknown", "unknown"

    low = u.lower()
    if any(
        x in low
        for x in (
            "bot",
            "spider",
            "crawler",
            "bingbot",
            "googlebot",
            "google-read-aloud",
            "aseg-",
            "genomecrawler",
            "petalbot",
            "yandex",
        )
    ):
        return "Bot/scanner", "bot"

    if "networkingextension" in low:
        return "iOS system fetch", "mobile"

    is_mobile = any(x in u for x in ("iPhone", "iPad", "Android", "Mobile", "CriOS"))

    if "edg/" in low:
        return "Edge", "mobile" if is_mobile else "desktop"
    if "firefox/" in low:
        return "Firefox", "mobile" if is_mobile else "desktop"
    if "samsungbrowser" in low:
        return "Samsung Internet", "mobile"
    if "chrome/" in low or "crios/" in low:
        return "Chrome", "mobile" if is_mobile else "desktop"
    if "safari" in low and "chrome" not in low and "chromium" not in low:
        if is_mobile or "iphone" in low or "ipad" in low:
            return "Safari iOS", "mobile"
        return "Safari macOS", "desktop"
    if "safari" in low:
        return "Safari (WebKit)", "mobile" if is_mobile else "desktop"

    return "Other", "mobile" if is_mobile else "desktop"


def parse_journal_line(line: str) -> Optional[JournalEvent]:
    line = line.strip()
    if not line:
        return None
    idx = line.find("ipv4_outage ")
    if idx >= 0:
        line = line[idx:]
    m = JOURNAL_RE.search(line)
    if m:
        return JournalEvent(**m.groupdict())
    jm = JOURNAL_JSON_RE.search(line)
    if jm:
        try:
            obj = json.loads(jm.group(1))
            ev = str(obj.get("event", ""))
            if ev in ("566", "recovery", "probe"):
                return JournalEvent(
                    event=ev,
                    token=str(obj.get("token", "")),
                    path=str(obj.get("path", "")),
                    client_ip=str(obj.get("client_ip", "")),
                    host=str(obj.get("host", "")),
                )
        except json.JSONDecodeError:
            pass
    return None


def parse_nginx_line(line: str) -> Optional[NginxRow]:
    line = line.rstrip("\n")
    if not line.strip():
        return None
    parts = line.split()
    if not parts:
        return None
    remote = parts[0]
    sm = NGINX_STATUS_RE.search(line)
    if not sm:
        return None
    status = int(sm.group(1))

    ua, vhost = "", ""
    um = UA_VHOST_RE.search(line)
    if um:
        ua, vhost = um.group(1), um.group(2)
    else:
        quoted = re.findall(r'"([^"]*)"', line)
        if len(quoted) >= 2:
            vhost = quoted[-1]
            for q in reversed(quoted[1:-1]):
                if q and q != "-" and len(q) > 3:
                    ua = q
                    break

    path = "/"
    quoted = re.findall(r'"([^"]*)"', line)
    if quoted:
        pm = NGINX_PATH_RE.match(quoted[0])
        if pm:
            path = normalize_path(pm.group(1))

    return NginxRow(
        remote_addr=remote,
        status=status,
        vhost=vhost,
        user_agent=ua,
        path=path,
        stack=connection_stack(remote),
    )


def nginx_date_marker(d: date) -> str:
    return d.strftime("%d/%b/%Y")


def discover_nginx_logs(drill_day: date) -> list[Path]:
    next_day = drill_day + timedelta(days=1)
    candidates = [
        NGINX_DIR / f"access.log-{next_day.strftime('%Y%m%d')}",
        NGINX_DIR / "access.log",
    ]
    found: list[Path] = []
    marker = nginx_date_marker(drill_day)
    for p in candidates:
        if p.is_file():
            found.append(p)
    if found:
        return found
    if NGINX_DIR.is_dir():
        for p in sorted(NGINX_DIR.glob("access.log*")):
            if p.is_file() and not p.name.endswith(".gz"):
                found.append(p)
    return found or candidates


def iter_nginx_lines(paths: Iterable[Path], drill_day: date) -> Iterator[str]:
    marker = nginx_date_marker(drill_day)
    for path in paths:
        if not path.is_file():
            continue
        if path.suffix == ".gz":
            try:
                import gzip

                with gzip.open(path, "rt", encoding="utf-8", errors="replace") as f:
                    for line in f:
                        if marker in line:
                            yield line
            except OSError:
                continue
        else:
            with open(path, encoding="utf-8", errors="replace") as f:
                for line in f:
                    if marker in line:
                        yield line


def read_journal(
    drill_day: date,
    service: str,
    journal_file: Optional[Path],
) -> list[JournalEvent]:
    start = f"{drill_day.isoformat()} 00:00:00"
    end = f"{(drill_day + timedelta(days=1)).isoformat()} 00:00:00"
    lines: list[str] = []
    if journal_file:
        lines = journal_file.read_text(encoding="utf-8", errors="replace").splitlines()
    else:
        cmd = [
            "journalctl",
            "-u",
            service,
            "--since",
            start,
            "--until",
            end,
            "-o",
            "cat",
            "--no-pager",
        ]
        try:
            proc = subprocess.run(cmd, capture_output=True, text=True, check=False)
        except FileNotFoundError:
            print(
                "journalctl not found; use --journal-file for offline input",
                file=sys.stderr,
            )
            return []
        if proc.returncode != 0:
            print(
                f"journalctl failed (exit {proc.returncode}); try sudo or --journal-file",
                file=sys.stderr,
            )
            if proc.stderr:
                print(proc.stderr.strip(), file=sys.stderr)
            return []
        lines = proc.stdout.splitlines()

    events: list[JournalEvent] = []
    for line in lines:
        if "ipv4_outage" not in line:
            continue
        ev = parse_journal_line(line)
        if ev:
            events.append(ev)
    return events


def lookup_geo_ip(ip: str, cache: dict[str, str], sleep_s: float) -> str:
    if ip in cache:
        return cache[ip]
    url = f"http://ip-api.com/json/{ip}?fields=status,countryCode"
    try:
        with urllib.request.urlopen(url, timeout=10) as resp:
            data = json.loads(resp.read().decode())
        cc = data.get("countryCode", "??") if data.get("status") == "success" else "??"
    except (urllib.error.URLError, json.JSONDecodeError, TimeoutError):
        cc = "??"
    cache[ip] = cc
    if sleep_s > 0:
        time.sleep(sleep_s)
    return cc


def format_table(headers: list[str], rows: list[list[str]]) -> str:
    widths = [len(h) for h in headers]
    for row in rows:
        for i, cell in enumerate(row):
            widths[i] = max(widths[i], len(cell))
    fmt = "  ".join(f"{{:{w}}}" for w in widths)
    out = [fmt.format(*headers), fmt.format(*["-" * w for w in widths])]
    for row in rows:
        out.append(fmt.format(*row))
    return "\n".join(out)


def render_report(
    drill_day: date,
    journal: list[JournalEvent],
    nginx_rows: list[NginxRow],
    main_host: str,
    geo_counts: Optional[dict[str, int]],
    geo_total: int,
    include_exempt: bool,
) -> str:
    lines: list[str] = []
    host = socket.gethostname()
    now = datetime.now().astimezone().strftime("%Y-%m-%d %H:%M:%S %Z")

    lines.append(f"# IPv4 outage drill report — {drill_day.isoformat()} UTC")
    lines.append(f"Generated: {now}  Host: {host}")
    lines.append("")

    ev566 = [e for e in journal if e.event == "566"]
    ev_recovery = [e for e in journal if e.event == "recovery"]
    tokens_566 = {e.token for e in ev566}
    tokens_recovery = {e.token for e in ev_recovery}
    matched = tokens_566 & tokens_recovery
    unique_blocked_ips = {e.client_ip for e in ev566}

    main_rows = [r for r in nginx_rows if r.vhost == main_host]
    exempt_main = [r for r in main_rows if is_exempt_path(r.path)]
    if include_exempt:
        main_primary = main_rows
    else:
        main_primary = [r for r in main_rows if not is_exempt_path(r.path)]

    main_566 = [r for r in main_primary if r.status == 566]
    total_main_566 = len(main_566)
    grand_total_requests = len(main_primary)

    lines.append("## Summary")
    lines.append(f"- 566 events (journald): {len(ev566)}")
    lines.append(f"- Recovery events (journald): {len(ev_recovery)}")
    lines.append(f"- Probe events (journald): {len([e for e in journal if e.event == 'probe'])}")
    lines.append(f"- Matched recovery tokens: {len(matched)}")
    if tokens_566:
        lines.append(
            f"- Token recovery rate: {pct(len(matched), len(tokens_566))} "
            f"({len(matched)} / {len(tokens_566)} unique 566 tokens)"
        )
    lines.append(f"- Unique IPv4 client_ip blocked (journald): {len(unique_blocked_ips)}")
    if not include_exempt:
        lines.append(
            f"- Main-host nginx requests excluded (exempt paths): {len(exempt_main)}"
        )
    lines.append(f"- Main-host nginx requests (blockable paths): {len(main_primary)}")
    lines.append(f"- Main-host nginx ×566 (blockable paths): {len(main_566)}")
    if main_primary:
        lines.append(
            f"- Main-host ×566 rate (blockable): {pct(len(main_566), len(main_primary))}"
        )
    lines.append("")

    # Stack table
    stack_stats: DefaultDict[str, BucketStats] = defaultdict(BucketStats)
    for r in main_primary:
        b = stack_stats[r.stack]
        b.requests += 1
        b.ips.add(r.remote_addr)
        if r.status == 566:
            b.blocked += 1

    lines.append("## Impact by connection stack (main host, blockable paths)")
    lines.append(
        "(Excludes /api/healthz, embed assets, and crawler paths — still HTTP 200 on IPv4 during the drill.)"
    )
    stack_rows = []
    for stack in ("ipv4", "ipv6"):
        b = stack_stats.get(stack, BucketStats())
        stack_rows.append(
            [
                stack.upper(),
                str(b.requests),
                str(b.blocked),
                pct(b.blocked, b.requests),
                str(len(b.ips)),
            ]
        )
    lines.append(
        format_table(
            ["Stack", "Requests", "×566", "%566", "Unique IPs"],
            stack_rows,
        )
    )
    lines.append("")

    # Merged client family table (IPv4 + IPv6 connection stacks)
    fam_stats: DefaultDict[str, FamilyStats] = defaultdict(FamilyStats)
    for r in main_primary:
        family, _ = classify_ua(r.user_agent)
        b = fam_stats[family]
        b.total += 1
        if r.stack == "ipv4":
            b.ipv4_requests += 1
            b.ipv4_ips.add(r.remote_addr)
        else:
            b.ipv6_requests += 1
            b.ipv6_ips.add(r.remote_addr)
        if r.status == 566:
            b.blocked += 1

    lines.append("## Impact by client family (main host, blockable paths)")
    fam_rows = []
    for family, b in sorted(fam_stats.items(), key=lambda x: (-x[1].blocked, -x[1].total)):
        fam_rows.append(
            [
                family,
                str(b.total),
                count_with_pct(b.ipv4_requests, b.total),
                count_with_pct(b.ipv6_requests, b.total),
                str(b.blocked),
                pct(b.blocked, b.total),
                str(len(b.ipv4_ips)),
                str(len(b.ipv6_ips)),
            ]
        )
    if not fam_rows:
        fam_rows.append(["(none)", "0", "0 (—)", "0 (—)", "0", "—", "0", "0"])
    lines.append(
        format_table(
            [
                "Family",
                "Total",
                "IPv4 reqs",
                "IPv6 reqs",
                "×566",
                "%566",
                "Unique IPv4",
                "Unique IPv6",
            ],
            fam_rows,
        )
    )
    lines.append("")

    # Exempt appendix
    if exempt_main and not include_exempt:
        lines.append("## Exempt paths (main host, excluded from tables above)")
        by_path: DefaultDict[str, Tuple[int, int]] = defaultdict(lambda: (0, 0))
        for r in exempt_main:
            p = normalize_path(r.path)
            t, b = by_path[p]
            by_path[p] = (t + 1, b + (1 if r.status == 566 else 0))
        ex_rows = [
            [p, str(t), str(b), pct(b, t)]
            for p, (t, b) in sorted(by_path.items(), key=lambda x: -x[1][0])
        ]
        lines.append(format_table(["Path", "Requests", "×566", "%566"], ex_rows))
        lines.append("")

    # Journal paths
    path_counts: DefaultDict[str, int] = defaultdict(int)
    for e in ev566:
        path_counts[e.path] += 1
    lines.append("## Top paths blocked (journald)")
    path_rows = []
    for p, c in sorted(path_counts.items(), key=lambda x: -x[1])[:25]:
        path_rows.append([p, str(c), pct(c, len(ev566))])
    lines.append(format_table(["Path", "Events", "% of 566"], path_rows))
    lines.append("")

    # Hourly — need timestamps from journal; journal -o cat lacks time. Note limitation.
    lines.append("## Hourly 566 volume (journald)")
    lines.append(
        "(Requires journal timestamps; re-run with `journalctl -o short-iso` piped to "
        "--journal-file for hourly breakdown, or use nginx time field.)"
    )
    hour_counts: DefaultDict[int, int] = defaultdict(int)
    for path in discover_nginx_logs(drill_day):
        marker = nginx_date_marker(drill_day)
        for raw in iter_nginx_lines([path], drill_day):
            if main_host not in raw or " 566 " not in raw:
                continue
            tm = re.search(r"\[(\d{2})/\w{3}/\d{4}:(\d{2}):", raw)
            if tm:
                hour_counts[int(tm.group(2))] += 1
    if hour_counts:
        lines.append("")
        for h in range(24):
            if h in hour_counts:
                lines.append(f"  {h:02d}:00 UTC  {hour_counts[h]}")
    lines.append("")

    # Probe vhosts
    probe_hosts = (
        f"ipv4.{main_host}",
        f"ipv6.{main_host}",
    )
    lines.append("## Probe reachability (nginx, drill day)")
    probe_rows = []
    for vh in probe_hosts:
        rows_vh = [r for r in nginx_rows if r.vhost == vh and r.status == 200]
        hz = [r for r in rows_vh if r.path == "/api/healthz"]
        probe_rows.append(
            [
                vh,
                str(len({r.remote_addr for r in rows_vh})),
                str(len(hz)),
                str(len({r.remote_addr for r in hz})),
            ]
        )
    lines.append(
        format_table(
            ["Vhost", "Unique IP (200)", "healthz 200", "Unique IP healthz"],
            probe_rows,
        )
    )
    lines.append("")

    if geo_counts is not None:
        lines.append("## Geography (unique blocked client_ip)")
        geo_rows = [
            [cc, str(n), pct(n, geo_total)]
            for cc, n in sorted(geo_counts.items(), key=lambda x: -x[1])[:30]
        ]
        lines.append(format_table(["Country", "Unique IPs", "% of blocked"], geo_rows))
        pacific = sum(geo_counts.get(c, 0) for c in PACIFIC_ISO2)
        lines.append(f"- Pacific ISO2 subset: {pacific} ({pct(pacific, geo_total)})")
        lines.append("")

    lines.append("## Notes / limitations")
    lines.append(
        "- Blockable paths exclude /api/healthz, embed assets, and crawler paths (IPv4 still gets HTTP 200)."
    )
    lines.append("- %566 = ×566 / total requests for that row (family or stack).")
    lines.append("- IPv4/IPv6 req columns show share of that family's requests per connection stack.")
    lines.append("- Dual-stack users appear in IPv4 and/or IPv6 columns depending on connection chosen.")
    lines.append("- Per-user IPv4→IPv6 recovery needs Retry-Over-IPv6-Recovery tokens in app logs.")
    lines.append("- journald path counts may include pre-deploy window; nginx UA is authoritative for browsers.")

    return "\n".join(lines)


def infer_drill_date() -> Optional[date]:
    today = datetime.utcnow().date()
    if today.day >= 6:
        return today.replace(day=6)
    first = today.replace(day=1)
    prev = first - timedelta(days=1)
    return prev.replace(day=6)


def main() -> int:
    parser = argparse.ArgumentParser(description="IPv4 outage post-drill report")
    parser.add_argument("--date", help="Drill UTC date YYYY-MM-DD")
    parser.add_argument("--service", default=DEFAULT_SERVICE)
    parser.add_argument("--main-host", default=DEFAULT_MAIN_HOST)
    parser.add_argument("--nginx-log", action="append", dest="nginx_logs", help="nginx access log path (repeatable)")
    parser.add_argument("--journal-file", type=Path, help="offline journal lines instead of journalctl")
    parser.add_argument("--geo", action="store_true", help="Resolve countries via ip-api.com (slow)")
    parser.add_argument("--geo-sleep", type=float, default=1.4, help="Seconds between geo API calls")
    parser.add_argument("--geo-cache", type=Path, help="TSV cache ip\\tcc")
    parser.add_argument(
        "--include-exempt",
        action="store_true",
        help="Include exempt paths in primary UA/stack tables",
    )
    parser.add_argument("-o", "--output", type=Path, help="Write report to file")
    args = parser.parse_args()

    if args.date:
        drill_day = date.fromisoformat(args.date)
    else:
        inferred = infer_drill_date()
        if not inferred:
            print("--date required", file=sys.stderr)
            return 1
        drill_day = inferred
        print(f"Using inferred drill date: {drill_day.isoformat()}", file=sys.stderr)

    journal = read_journal(drill_day, args.service, args.journal_file)
    nginx_paths = [Path(p) for p in args.nginx_logs] if args.nginx_logs else discover_nginx_logs(drill_day)

    nginx_rows: list[NginxRow] = []
    missing_logs = []
    for path in nginx_paths:
        if not path.is_file():
            missing_logs.append(str(path))
            continue
        for raw in iter_nginx_lines([path], drill_day):
            row = parse_nginx_line(raw)
            if row:
                nginx_rows.append(row)

    if missing_logs and not nginx_rows:
        print(f"warning: nginx logs not found: {', '.join(missing_logs)}", file=sys.stderr)

    geo_counts: Optional[dict[str, int]] = None
    geo_total = 0
    if args.geo:
        blocked_ips = {e.client_ip for e in journal if e.event == "566"}
        cache: dict[str, str] = {}
        if args.geo_cache and args.geo_cache.is_file():
            for line in args.geo_cache.read_text().splitlines():
                parts = line.split("\t", 1)
                if len(parts) == 2:
                    cache[parts[0]] = parts[1]
        geo_counts = defaultdict(int)
        print(f"Geo lookup: {len(blocked_ips)} IPs...", file=sys.stderr)
        cache_path = args.geo_cache or Path(f"/tmp/ipv4-outage-geo-{drill_day.isoformat()}.tsv")
        with open(cache_path, "a+", encoding="utf-8") as cache_f:
            cache_f.seek(0)
            for line in cache_f:
                parts = line.strip().split("\t", 1)
                if len(parts) == 2:
                    cache[parts[0]] = parts[1]
            for ip in sorted(blocked_ips):
                if ip not in cache:
                    cc = lookup_geo_ip(ip, cache, args.geo_sleep)
                    cache_f.write(f"{ip}\t{cc}\n")
                    cache_f.flush()
                geo_counts[cache[ip]] += 1
        geo_total = len(blocked_ips)

    report = render_report(
        drill_day,
        journal,
        nginx_rows,
        args.main_host,
        geo_counts,
        geo_total,
        args.include_exempt,
    )

    if args.output:
        args.output.write_text(report, encoding="utf-8")
        print(f"Wrote {args.output}", file=sys.stderr)
    else:
        print(report)
    return 0


if __name__ == "__main__":
    sys.exit(main())
