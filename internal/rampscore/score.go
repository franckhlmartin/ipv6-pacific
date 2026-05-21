package rampscore

import "strings"

// DMARCScorePct maps DMARC state and policies to a 0-100 ramp value.
// hasData is false for error/unknown measurement (UI grey).
func DMARCScorePct(state, policy, subdomainPolicy string) (pct float64, hasData bool) {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "absent":
		return 0, true
	case "present":
		effective := maxPolicyRank(policy, effectiveSubdomainPolicy(policy, subdomainPolicy))
		switch effective {
		case 2:
			return 100, true
		case 1:
			return 75, true
		default:
			return 25, true
		}
	default:
		return 0, false
	}
}

func effectiveSubdomainPolicy(p, sp string) string {
	sp = strings.ToLower(strings.TrimSpace(sp))
	if sp != "" {
		return sp
	}
	return strings.ToLower(strings.TrimSpace(p))
}

func maxPolicyRank(p, sp string) int {
	return max(policyRank(p), policyRank(sp))
}

func policyRank(p string) int {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "reject":
		return 2
	case "quarantine":
		return 1
	case "none":
		return 0
	default:
		return 0
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// RPKIScorePct maps sampled prefix validation counts to 0-100:
// share of prefixes RIPEstat marked valid (invalid and unknown count as non-valid).
// Worst-case label is separate (RPKIWorstStatus) for the status column.
// hasData is false when total==0 or err is set.
func RPKIScorePct(valid, invalid, unknown, total int, err string) (pct float64, hasData bool) {
	if strings.TrimSpace(err) != "" || total <= 0 {
		return 0, false
	}
	return 100 * float64(valid) / float64(total), true
}

// RPKIWorstStatus returns advocacy label: invalid > unknown > valid.
func RPKIWorstStatus(valid, invalid, unknown int) string {
	if invalid > 0 {
		return "invalid"
	}
	if unknown > 0 && valid == 0 {
		return "unknown"
	}
	if valid > 0 && invalid == 0 {
		return "valid"
	}
	if unknown > 0 {
		return "unknown"
	}
	return ""
}
