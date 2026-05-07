package checks

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/pacific-monitor/pacific-monitor/internal/model"
)

func checkMail(ctx context.Context, apex string, cfg Config) (model.ServiceColumn, error) {
	col := model.ServiceColumn{Location: "-", Display: "[0] -/-/- [-]"}

	c := new(dns.Client)
	c.Timeout = cfg.DNSResolveTimeout

	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(apex), dns.TypeMX)
	msg.RecursionDesired = true

	r, _, err := c.ExchangeContext(ctx, msg, "8.8.8.8:53")
	if err != nil || r == nil {
		col.Display = fmt.Sprintf("[0] -/-/- [-]")
		col.IntentionallyNA = true
		col.Class = model.DeployUnknown
		return col, nil
	}

	var mxHosts []string
	for _, a := range r.Answer {
		if mx, ok := a.(*dns.MX); ok {
			h := strings.TrimSuffix(strings.ToLower(mx.Mx), ".")
			mxHosts = append(mxHosts, h)
		}
	}
	if len(mxHosts) == 0 {
		col.Display = "[0] -/-/- [-]"
		col.IntentionallyNA = true
		col.Class = model.DeployUnknown
		return col, nil
	}

	resolver := net.Resolver{PreferGo: true}
	var locTags []string
	for _, h := range mxHosts {
		locTags = append(locTags, classifyLocation(h, apex))
	}
	col.Location = mergeLocationTags(locTags)

	v4cfg, v4reach, v4op := 0, 0, 0
	v6cfg, v6reach, v6op := 0, 0, 0

	for _, host := range uniqueStrings(mxHosts) {
		addrs, err := resolver.LookupIPAddr(ctx, host)
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
			v4reach++
			if smtpEhlo(ctx, net.JoinHostPort(ip.String(), "25"), cfg.SMTPTimeout) {
				v4op++
			}
		}
		for _, ip := range v6s {
			v6reach++
			if smtpEhlo(ctx, net.JoinHostPort(ip.String(), "25"), cfg.SMTPTimeout) {
				v6op++
			}
		}
	}

	col.IPv4 = model.ServiceMetrics{Configured: v4cfg, Reachable: v4reach, Operational: v4op}
	col.IPv6 = model.ServiceMetrics{Configured: v6cfg, Reachable: v6reach, Operational: v6op}
	// Show v4 and v6 SMTP (EHLO:25) separately — v6 addrs can exist while port 25 is closed/filtered (e.g. Microsoft).
	col.Display = fmt.Sprintf("[%d MX] v4 smtp %d/%d/%d v6 smtp %d/%d/%d [%s]",
		len(mxHosts), v4cfg, v4reach, v4op, v6cfg, v6reach, v6op, col.Location)
	col.Class = classifyService(col.IPv4, col.IPv6)
	return col, nil
}

func smtpEhlo(ctx context.Context, addr string, timeout time.Duration) bool {
	ctx2, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var d net.Dialer
	conn, err := d.DialContext(ctx2, "tcp", addr)
	if err != nil {
		return false
	}
	defer conn.Close()

	br := bufio.NewReader(conn)
	deadline, _ := ctx2.Deadline()
	_ = conn.SetDeadline(deadline)

	line, err := br.ReadString('\n')
	if err != nil || !strings.HasPrefix(line, "220") {
		return false
	}
	fmt.Fprintf(conn, "EHLO pacific-monitor.local\r\n")
	for i := 0; i < 20; i++ {
		l, err := br.ReadString('\n')
		if err != nil {
			return false
		}
		if strings.HasPrefix(l, "250 ") {
			return true
		}
	}
	return false
}

func mailLegendExplanation() LegendCheckExplanation {
	return LegendCheckExplanation{
		ID:           "mail",
		Title:        "Mail",
		Format:       "[MX count] v4 smtp configured/reachable/operational v6 smtp configured/reachable/operational [location]",
		PlainMeaning: "Shows MX footprint and SMTP health per IP family. Operational means an SMTP banner and EHLO exchange succeeded on port 25.",
		Notes: []string{
			"v4 and v6 SMTP are measured separately.",
			"MX presence without successful SMTP handshake can still produce non-operational results.",
		},
	}
}
