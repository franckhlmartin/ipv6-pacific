package apnicstats

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

var (
	reDrawTableBlock = regexp.MustCompile(`(?s)function\s+drawTable\s*\(\s*\)\s*\{.*?arrayToDataTable\s*\(\s*\[(.*?)\]\s*\)\s*;`)
	reASNAnchor      = regexp.MustCompile(`>AS(\d+)</a>`)
	reDataRow        = regexp.MustCompile(`(?m)^\s*\[.*?>(AS\d+)</a>","(.*?)"\s*,\s*\{v:\s*([0-9.]+).*?\}\s*,\s*\{v:\s*([0-9.]+).*?\}\s*,\s*(\d+)\s*\]`)
	reMetricV        = regexp.MustCompile(`\{v:\s*([0-9.]+)`)
)

// ParseCountryASNTableHTML extracts per-ASN rows from the drawTable() block on stats.labs.apnic.net/ipv6/{CC}.
func ParseCountryASNTableHTML(r io.Reader) ([]ASNRow, error) {
	body, err := io.ReadAll(io.LimitReader(r, 15<<20))
	if err != nil {
		return nil, err
	}
	return ParseCountryASNTableHTMLString(string(body))
}

// ParseCountryASNTableHTMLString parses embedded drawTable arrayToDataTable data.
func ParseCountryASNTableHTMLString(html string) ([]ASNRow, error) {
	m := reDrawTableBlock.FindStringSubmatch(html)
	if len(m) < 2 {
		return nil, fmt.Errorf("apnicstats: drawTable block not found")
	}
	block := m[1]
	if !strings.Contains(block, "IPv6 Preferred") {
		return nil, fmt.Errorf("apnicstats: IPv6 Preferred header not found")
	}

	var out []ASNRow
	for _, rowMatch := range reDataRow.FindAllStringSubmatch(block, -1) {
		if len(rowMatch) < 6 {
			continue
		}
		asnStr := strings.TrimSpace(rowMatch[1])
		asnNum, err := strconv.Atoi(strings.TrimPrefix(asnStr, "AS"))
		if err != nil || asnNum <= 0 {
			continue
		}
		name := strings.TrimSpace(rowMatch[2])
		capable, _ := strconv.ParseFloat(rowMatch[3], 64)
		preferred, _ := strconv.ParseFloat(rowMatch[4], 64)
		samples, _ := strconv.Atoi(rowMatch[5])
		out = append(out, ASNRow{
			ASN:              asnStr,
			ASNNumber:        asnNum,
			Name:             name,
			IPv6CapablePct:   capable,
			IPv6PreferredPct: preferred,
			Samples:          samples,
		})
	}

	if len(out) == 0 {
		// Fallback: line-by-line scan for rows with AS<number> anchors.
		out = parseRowsLineScan(block)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("apnicstats: no ASN rows parsed")
	}
	return out, nil
}

func parseRowsLineScan(block string) []ASNRow {
	var out []ASNRow
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, `["`) || !strings.Contains(line, "AS") {
			continue
		}
		asnM := reASNAnchor.FindStringSubmatch(line)
		if len(asnM) < 2 {
			continue
		}
		asnNum, err := strconv.Atoi(asnM[1])
		if err != nil || asnNum <= 0 {
			continue
		}
		// Name is the quoted string immediately after the anchor cell.
		const anchorEnd = `</a>","`
		idx := strings.Index(line, anchorEnd)
		if idx < 0 {
			continue
		}
		namePart := line[idx+len(anchorEnd):]
		endName := strings.Index(namePart, `"`)
		if endName < 0 {
			continue
		}
		name := strings.TrimSpace(namePart[:endName])

		metrics := reMetricV.FindAllStringSubmatch(line, -1)
		if len(metrics) < 2 {
			continue
		}
		capable, _ := strconv.ParseFloat(metrics[0][1], 64)
		preferred, _ := strconv.ParseFloat(metrics[1][1], 64)

		samples := 0
		if idx := strings.LastIndex(line, ","); idx >= 0 {
			tail := strings.TrimRight(strings.TrimSpace(line[idx+1:]), "]")
			if n, err := strconv.Atoi(tail); err == nil {
				samples = n
			}
		}

		out = append(out, ASNRow{
			ASN:              fmt.Sprintf("AS%d", asnNum),
			ASNNumber:        asnNum,
			Name:             name,
			IPv6CapablePct:   capable,
			IPv6PreferredPct: preferred,
			Samples:          samples,
		})
	}
	return out
}
