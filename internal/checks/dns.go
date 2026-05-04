package checks

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/pacific-monitor/pacific-monitor/internal/model"
)

func checkDNS(ctx context.Context, apex string, cfg Config) (model.ServiceColumn, []string, error) {
	col := model.ServiceColumn{Location: "-", Display: "[0] -/-/- [-]"}
	var locTags []string

	resolver := net.Resolver{PreferGo: true}

	ctxLookup, cancel := context.WithTimeout(ctx, cfg.DNSResolveTimeout)
	defer cancel()

	c := new(dns.Client)
	c.Timeout = cfg.DNSResolveTimeout

	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(apex), dns.TypeNS)
	msg.RecursionDesired = true

	r, _, err := c.ExchangeContext(ctxLookup, msg, "8.8.8.8:53")
	if err != nil || r == nil {
		col.Display = fmt.Sprintf("[S] error: %v", err)
		col.Class = model.DeployUnknown
		return col, locTags, err
	}

	var nsHosts []string
	for _, a := range r.Answer {
		if ns, ok := a.(*dns.NS); ok {
			h := strings.TrimSuffix(strings.ToLower(ns.Ns), ".")
			nsHosts = append(nsHosts, h)
			locTags = append(locTags, classifyLocation(h, apex))
		}
	}
	if len(nsHosts) == 0 {
		col.Display = "[0] -/-/- [-]"
		col.Class = model.DeployUnknown
		col.Location = "-"
		return col, locTags, nil
	}

	col.Location = mergeLocationTags(locTags)

	v4cfg, v4reach, v4op := 0, 0, 0
	v6cfg, v6reach, v6op := 0, 0, 0

	for _, host := range uniqueStrings(nsHosts) {
		addrs, err := resolver.LookupIPAddr(ctxLookup, host)
		if err != nil {
			continue
		}
		var v4s, v6s []net.IP
		for _, a := range addrs {
			if a.IP.To4() != nil {
				v4s = append(v4s, a.IP)
			} else if a.IP.To16() != nil && a.IP.To4() == nil {
				v6s = append(v6s, a.IP)
			}
		}
		v4s = uniqueIPs(v4s)
		v6s = uniqueIPs(v6s)

		v4cfg += len(v4s)
		v6cfg += len(v6s)

		for _, ip := range v4s {
			if udpOK(ctx, c, ip.String()+":53", apex, dns.TypeSOA) {
				v4reach++
				v4op++
			}
		}
		for _, ip := range v6s {
			if udpOK(ctx, c, "["+ip.String()+"]:53", apex, dns.TypeSOA) {
				v6reach++
				v6op++
			}
		}
	}

	col.IPv4 = model.ServiceMetrics{Configured: v4cfg, Reachable: v4reach, Operational: v4op}
	col.IPv6 = model.ServiceMetrics{Configured: v6cfg, Reachable: v6reach, Operational: v6op}
	un := uniqueStrings(nsHosts)
	col.Display = fmt.Sprintf("[%d] %d/%d/%d [%s]", len(un), v6cfg, v6reach, v6op, col.Location)
	col.Class = classifyService(col.IPv4, col.IPv6)

	return col, locTags, nil
}

func udpOK(ctx context.Context, c *dns.Client, serverAddr, qname string, qtype uint16) bool {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(qname), qtype)
	msg.RecursionDesired = false
	ctx2, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	r, _, err := c.ExchangeContext(ctx2, msg, serverAddr)
	return err == nil && r != nil && (r.Rcode == dns.RcodeSuccess || r.Rcode == dns.RcodeNameError)
}

func uniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func uniqueIPs(in []net.IP) []net.IP {
	seen := map[string]struct{}{}
	var out []net.IP
	for _, ip := range in {
		k := ip.String()
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, ip)
	}
	return out
}
