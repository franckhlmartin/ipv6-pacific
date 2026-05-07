package checks

import (
	"context"

	"github.com/miekg/dns"
	"github.com/pacific-monitor/pacific-monitor/internal/model"
)

func checkDNSSEC(ctx context.Context, apex string, cfg Config) model.DNSSECColumn {
	c := new(dns.Client)
	c.Timeout = cfg.DNSResolveTimeout

	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(apex), dns.TypeDNSKEY)
	msg.SetEdns0(4096, false)

	r, _, err := c.ExchangeContext(ctx, msg, "8.8.8.8:53")
	if err != nil || r == nil {
		return model.DNSSECColumn{State: "error", Summary: "?/?/?", Display: err.Error()}
	}
	hasDNSKEY := false
	for _, a := range r.Answer {
		if _, ok := a.(*dns.DNSKEY); ok {
			hasDNSKEY = true
			break
		}
	}
	if !hasDNSKEY {
		return model.DNSSECColumn{State: "unsigned", Summary: "U/-/-", Display: "U/-/-"}
	}
	// Full chain validation deferred; mark signed for v1.
	return model.DNSSECColumn{State: "signed", Summary: "S/?/?", Display: "S/?/? (partial)"}
}

func dnssecLegendExplanation() LegendCheckExplanation {
	return LegendCheckExplanation{
		ID:           "dnssec",
		Title:        "DNSSEC",
		Format:       "S/?/? (partial), U/-/-, or error text",
		PlainMeaning: "Shows whether DNSKEY data is observed at the apex (signed), absent (unsigned), or unavailable due to lookup error.",
		Notes: []string{
			"S/?/? (partial) indicates partial DNSSEC signal only.",
			"Full chain validation is not yet performed by this checker.",
		},
	}
}
