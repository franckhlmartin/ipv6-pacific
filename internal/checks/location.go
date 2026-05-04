package checks

import (
	"strings"
)

// classifyLocation returns I (inside), P (parent), O (outside), or M (mixed) comparing
// service hostname to the domain under test apex (e.g. gov.fj).
func classifyLocation(serviceHost, apex string) string {
	serviceHost = strings.TrimSuffix(strings.TrimSuffix(strings.ToLower(serviceHost), "."), ".")
	apex = strings.TrimSuffix(strings.TrimSuffix(strings.ToLower(apex), "."), ".")

	if serviceHost == apex || strings.HasSuffix(serviceHost, "."+apex) {
		return "I"
	}

	// Parent zone: direct child label under apex — heuristic for P
	if idx := strings.LastIndex(serviceHost, "."); idx > 0 {
		parent := serviceHost[idx+1:]
		if parent == apex {
			return "P"
		}
	}

	labels := strings.Split(serviceHost, ".")
	for i := range labels {
		suffix := strings.Join(labels[i:], ".")
		if suffix == apex {
			return "P"
		}
	}

	return "O"
}

func mergeLocationTags(tags []string) string {
	if len(tags) == 0 {
		return "-"
	}
	set := map[string]struct{}{}
	for _, t := range tags {
		set[t] = struct{}{}
	}
	if len(set) == 1 {
		for k := range set {
			return k
		}
	}
	has := func(x string) bool {
		_, ok := set[x]
		return ok
	}
	if has("I") && (has("O") || has("P")) {
		return "M"
	}
	if has("P") && has("O") {
		return "M"
	}
	return "M"
}
