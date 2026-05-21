package checks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pacific-monitor/pacific-monitor/internal/model"
)

// DomainMeta is optional table metadata from YAML.
type DomainMeta struct {
	Organization string
	Sector       string
	// WebURL optional full HTTPS URL (or host) tried before https://{apex}/ and https://www.{apex}/.
	WebURL string
}

// RunDomain executes DNS, Mail, Web, and DNSSEC checks for a registered domain (apex).
func RunDomain(ctx context.Context, apex string, cfg Config, meta DomainMeta) model.DomainResult {
	apex = strings.TrimSuffix(strings.TrimSpace(strings.ToLower(apex)), ".")
	ctx, cancel := context.WithTimeout(ctx, cfg.DomainDeadline)
	defer cancel()

	res := model.DomainResult{
		Domain:       apex,
		Organization: meta.Organization,
		Sector:       meta.Sector,
	}

	dnsCol, _, err := checkDNS(ctx, apex, cfg)
	if err != nil {
		res.Error = fmt.Sprintf("dns: %v", err)
	}
	res.DNS = dnsCol
	logStep(cfg, "DNS", fmt.Sprintf("DNSResolveTimeout=%s (NS/exchange + resolver); UDP SOA probes ≤3s", dur(cfg.DNSResolveTimeout)), summarizeDNS(dnsCol, err))

	mailCol, err := checkMail(ctx, apex, cfg)
	if err != nil {
		if res.Error == "" {
			res.Error = fmt.Sprintf("mail: %v", err)
		}
	}
	res.Mail = mailCol
	logStep(cfg, "Mail", mailTimeoutDesc(cfg), summarizeMail(mailCol, err))

	webCol, _, err := checkWeb(ctx, apex, cfg, meta.WebURL)
	if err != nil && res.Error == "" {
		res.Error = fmt.Sprintf("web: %v", err)
	}
	res.Web = webCol
	logStep(cfg, "Web", fmt.Sprintf("HTTPTimeout=%s (HTTPS discover + per-family GET)", dur(cfg.HTTPTimeout)), summarizeWeb(webCol, err))

	res.DNSSEC = checkDNSSEC(ctx, apex, cfg)
	logStep(cfg, "DNSSEC", fmt.Sprintf("DNSResolveTimeout=%s (DNSKEY query)", dur(cfg.DNSResolveTimeout)), summarizeDNSSEC(res.DNSSEC))

	res.DMARC = checkDMARC(ctx, apex, cfg)
	logStep(cfg, "DMARC", fmt.Sprintf("DNSResolveTimeout=%s (_dmarc TXT)", dur(cfg.DNSResolveTimeout)), summarizeDMARC(res.DMARC))

	res.RollupClass = Rollup(res.DNS.Class, res.Mail.Class, res.Web.Class)
	return res
}

func dur(d time.Duration) string {
	if d == 0 {
		return "0s"
	}
	return d.String()
}

func logStep(cfg Config, phase, timeoutDesc, summary string) {
	if cfg.LogStep == nil {
		return
	}
	cfg.LogStep(phase, timeoutDesc, summary)
}

func summarizeDNS(col model.ServiceColumn, err error) string {
	s := fmt.Sprintf("class=%s location=%s display=%q", col.Class, col.Location, col.Display)
	if err != nil {
		s += fmt.Sprintf(" error=%v", err)
	}
	return s
}

func summarizeMail(col model.ServiceColumn, err error) string {
	s := fmt.Sprintf("class=%s location=%s display=%q intentionally_na=%v", col.Class, col.Location, col.Display, col.IntentionallyNA)
	if err != nil {
		s += fmt.Sprintf(" error=%v", err)
	}
	return s
}

func summarizeWeb(col model.ServiceColumn, err error) string {
	s := fmt.Sprintf("class=%s location=%s display=%q", col.Class, col.Location, col.Display)
	if err != nil {
		s += fmt.Sprintf(" error=%v", err)
	}
	return s
}

func summarizeDNSSEC(col model.DNSSECColumn) string {
	return fmt.Sprintf("state=%s summary=%s display=%q", col.State, col.Summary, col.Display)
}

func summarizeDMARC(col model.DMARCColumn) string {
	return fmt.Sprintf("state=%s policy=%s sp=%s score_pct=%.0f display=%q", col.State, col.Policy, col.SubdomainPolicy, col.ScorePct, col.Display)
}

func mailTimeoutDesc(cfg Config) string {
	return fmt.Sprintf("MX NS query client=%s; SMTP EHLO per IP=%s", dur(cfg.DNSResolveTimeout), dur(cfg.SMTPTimeout))
}
