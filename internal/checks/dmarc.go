package checks

import (
	"context"
	"errors"
	"strings"

	"github.com/miekg/dns"
	"github.com/pacific-monitor/pacific-monitor/internal/model"
	"github.com/pacific-monitor/pacific-monitor/internal/rampscore"
)

func checkDMARC(ctx context.Context, apex string, cfg Config) model.DMARCColumn {
	name := "_dmarc." + apex
	c := new(dns.Client)
	c.Timeout = cfg.DNSResolveTimeout

	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(name), dns.TypeTXT)
	msg.RecursionDesired = true

	r, _, err := c.ExchangeContext(ctx, msg, "8.8.8.8:53")
	if err != nil {
		if dnsNotFound(err) {
			return absentDMARC()
		}
		return dmarcLookupError(err)
	}
	if r == nil {
		return dmarcLookupError(errors.New("empty dns response"))
	}
	if r.Rcode == dns.RcodeNameError {
		return absentDMARC()
	}
	if r.Rcode != dns.RcodeSuccess {
		return model.DMARCColumn{
			State:   "error",
			Display: dns.RcodeToString[r.Rcode],
		}
	}

	var dmarcRecords []string
	for _, a := range r.Answer {
		if txt, ok := a.(*dns.TXT); ok {
			s := strings.Join(txt.Txt, "")
			if strings.Contains(strings.ToUpper(s), "V=DMARC1") {
				dmarcRecords = append(dmarcRecords, s)
			}
		}
	}

	if len(dmarcRecords) == 0 {
		return absentDMARC()
	}

	raw := dmarcRecords[0]
	tags := parseDMARCTags(raw)
	rawP := tags["p"]
	rawSP := tags["sp"]
	policy := normalizePolicy(rawP)
	if policy == "" {
		return model.DMARCColumn{State: "error", Display: "invalid p="}
	}
	spEffective := effectiveSP(policy, rawSP)

	col := model.DMARCColumn{
		State:           "present",
		Exists:          true,
		Policy:          policy,
		SubdomainPolicy: spEffective,
		RawP:            rawP,
		RawSP:           rawSP,
	}
	col.Display = dmarcDisplay(policy, rawSP, spEffective)
	col.ScorePct, _ = rampscore.DMARCScorePct(col.State, col.Policy, col.SubdomainPolicy)
	return col
}

func absentDMARC() model.DMARCColumn {
	col := model.DMARCColumn{State: "absent", Display: "No record"}
	col.ScorePct, _ = rampscore.DMARCScorePct("absent", "", "")
	return col
}

func dmarcLookupError(err error) model.DMARCColumn {
	msg := "lookup failed"
	if err != nil {
		msg = err.Error()
	}
	return model.DMARCColumn{State: "error", Display: msg}
}

// dnsNotFound reports whether err indicates the _dmarc name does not exist.
func dnsNotFound(err error) bool {
	if err == nil {
		return false
	}
	u := strings.ToUpper(err.Error())
	return strings.Contains(u, "NXDOMAIN") || strings.Contains(u, "NO SUCH HOST")
}

func parseDMARCTags(record string) map[string]string {
	out := make(map[string]string)
	for _, part := range strings.Split(record, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		i := strings.IndexByte(part, '=')
		if i <= 0 {
			continue
		}
		k := strings.ToLower(strings.TrimSpace(part[:i]))
		v := strings.TrimSpace(part[i+1:])
		out[k] = v
	}
	return out
}

func normalizePolicy(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "none", "quarantine", "reject":
		return strings.ToLower(strings.TrimSpace(p))
	default:
		return ""
	}
}

func effectiveSP(orgPolicy, rawSP string) string {
	if sp := normalizePolicy(rawSP); sp != "" {
		return sp
	}
	return orgPolicy
}

func dmarcDisplay(policy, rawSP, spEffective string) string {
	if rawSP != "" && strings.ToLower(rawSP) != spEffective {
		return policy + " / " + spEffective
	}
	if spEffective != "" && spEffective != policy {
		return "p=" + policy + ", sp=" + spEffective
	}
	return policy
}

func dmarcLegendExplanation() LegendCheckExplanation {
	return LegendCheckExplanation{
		ID:           "dmarc",
		Title:        "DMARC",
		Format:       "policy or p=…, sp=…",
		PlainMeaning: "Published DMARC policy at _dmarc.{domain} (DNS TXT). Shows organizational policy (p=) and subdomain policy (sp=) when set.",
		Notes: []string{
			"No record (NXDOMAIN or no DMARC TXT) scores 0% on the ramp (red).",
			"p=none scores 25%; quarantine 75%; reject 100%. Stricter of p and sp drives the color.",
			"Lookup failures (timeout, SERVFAIL) show grey — not scored as absent.",
			"Does not measure SPF/DKIM pass rates — policy publication only.",
		},
	}
}
