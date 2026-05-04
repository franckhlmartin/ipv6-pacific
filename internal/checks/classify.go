package checks

import "github.com/pacific-monitor/pacific-monitor/internal/model"

// classifyService maps per-service IPv4/IPv6 metrics to orange/blue/green semantics.
func classifyService(v4, v6 model.ServiceMetrics) model.DeployClass {
	v4op := v4.Operational > 0
	v6op := v6.Operational > 0
	v4cfg := v4.Configured > 0
	v6cfg := v6.Configured > 0

	if v4cfg && v4op && v6cfg && v6op {
		return model.DeployDual
	}
	if v6cfg && v6op && !(v4cfg && v4op) {
		// Meaningful IPv6 operations without requiring legacy IPv4 path.
		return model.DeployIPv6Only
	}
	if v4cfg && v4op && (!v6cfg || !v6op) {
		return model.DeployIPv4Only
	}
	if !v4cfg && !v6cfg {
		return model.DeployUnknown
	}
	return model.DeployIPv4Only
}

// Rollup picks the "worst" column for advocacy: IPv4-only beats dual beats IPv6-only.
func Rollup(dns, mail, web model.DeployClass) model.DeployClass {
	rank := map[model.DeployClass]int{
		model.DeployIPv6Only: 0,
		model.DeployDual:     1,
		model.DeployUnknown:  2,
		model.DeployIPv4Only: 3,
	}
	worst := model.DeployIPv6Only
	bestRank := -1
	for _, c := range []model.DeployClass{dns, mail, web} {
		if rank[c] > bestRank {
			bestRank = rank[c]
			worst = c
		}
	}
	return worst
}
