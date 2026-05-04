package summary

import "github.com/pacific-monitor/pacific-monitor/internal/model"

// FromDomains builds header summary counts for a country page (Afrinic-style strip).
func FromDomains(domains []model.DomainResult) model.DeploymentSummary {
	var s model.DeploymentSummary
	for _, d := range domains {
		switch d.RollupClass {
		case model.DeployIPv4Only:
			s.IPv4Only++
		case model.DeployDual:
			s.Dual++
		case model.DeployIPv6Only:
			s.IPv6Only++
		}
		if d.DNS.IPv4.Operational+d.DNS.IPv6.Operational > 0 {
			s.DNS++
		}
		if d.Mail.IPv4.Operational+d.Mail.IPv6.Operational > 0 {
			s.Mail++
		}
		if d.Web.IPv4.Operational+d.Web.IPv6.Operational > 0 {
			s.Web++
		}
		if d.DNSSEC.State == "signed" || d.DNSSEC.State == "unsigned" {
			s.DNSSEC++
		}
	}
	return s
}
