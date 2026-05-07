package scoring

import (
	"strings"

	"github.com/pacific-monitor/pacific-monitor/internal/model"
)

// MaxRowScore is the maximum points returned by RowScore (four columns × 1).
const MaxRowScore = 4

// RowScore sums per-column points for DNS, Mail, Web, and DNSSEC (0–4).
// Deploy classes: ipv4_only and unknown → 0 (orange / grey), dual_stack → 1 (blue), ipv6_only → 1 (green).
//
// DNSSEC points (0/1 scale as a “column”):
//   - signed (DNSKEY present at apex; chain check deferred in collector) → 1 / green
//   - unsigned → 0 / orange
//   - error or unknown state → 0 / grey
func RowScore(d model.DomainResult) int {
	return pointsClass(d.DNS.Class) +
		pointsClass(d.Mail.Class) +
		pointsClass(d.Web.Class) +
		pointsDNSSEC(d.DNSSEC)
}

func pointsClass(c model.DeployClass) int {
	switch c {
	case model.DeployIPv6Only, model.DeployDual:
		return 1
	case model.DeployIPv4Only, model.DeployUnknown:
		return 0
	default:
		return 0
	}
}

func pointsDNSSEC(col model.DNSSECColumn) int {
	switch strings.ToLower(col.State) {
	case "signed":
		return 1
	case "unsigned", "error", "":
		return 0
	default:
		return 0
	}
}

// DNSSECCellClass returns the same DeployClass-style suffix used by DNS/Mail/Web cells
// (cell--ipv4_only / cell--dual_stack / cell--ipv6_only / cell--unknown) for consistent coloring.
func DNSSECCellClass(col model.DNSSECColumn) string {
	switch strings.ToLower(col.State) {
	case "signed":
		return string(model.DeployIPv6Only)
	case "unsigned":
		return string(model.DeployIPv4Only)
	case "error", "":
		return string(model.DeployUnknown)
	default:
		return string(model.DeployUnknown)
	}
}

// EconomyDeploymentScorePct is the economy-wide deployment score as a percentage (0–100).
// For each domain it uses RowScore (0–MaxRowScore); the result is the average fraction of
// the maximum, times 100, so economies with different domain counts remain comparable.
func EconomyDeploymentScorePct(domains []model.DomainResult) float64 {
	n := len(domains)
	if n == 0 {
		return 0
	}
	var sum int
	for _, d := range domains {
		sum += RowScore(d)
	}
	return 100 * float64(sum) / float64(MaxRowScore*n)
}
