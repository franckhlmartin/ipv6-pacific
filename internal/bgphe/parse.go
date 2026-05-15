package bgphe

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"golang.org/x/net/html"

	"github.com/pacific-monitor/pacific-monitor/internal/model"
)

// ParseCountryNetworksHTML extracts network rows from HE country listing HTML.
func ParseCountryNetworksHTML(r io.Reader) ([]model.BGPHENetworkRow, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	for _, tbl := range collectTag(doc, "table") {
		rows, err := parseNetworksTable(tbl)
		if err != nil {
			continue
		}
		if len(rows) > 0 {
			return rows, nil
		}
	}
	return nil, fmt.Errorf("bgphe: no networks table found")
}

func collectTag(n *html.Node, tag string) []*html.Node {
	var out []*html.Node
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == tag {
			out = append(out, node)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return out
}

func parseNetworksTable(table *html.Node) ([]model.BGPHENetworkRow, error) {
	trs := collectTag(table, "tr")
	if len(trs) < 2 {
		return nil, fmt.Errorf("bgphe: table too small")
	}

	var headerIdx int
	var colASN, colName, colV4, colV6 = -1, -1, -1, -1
	for i, tr := range trs {
		ths := collectTag(tr, "th")
		if len(ths) == 0 {
			continue
		}
		headers := make([]string, len(ths))
		for j, th := range ths {
			headers[j] = normalizeHeaderText(th)
		}
		asn, name, v4, v6 := mapColumns(headers)
		if asn >= 0 && name >= 0 && v4 >= 0 && v6 >= 0 {
			headerIdx = i
			colASN, colName, colV4, colV6 = asn, name, v4, v6
			break
		}
	}
	if colASN < 0 {
		return nil, fmt.Errorf("bgphe: header row not recognized")
	}

	var out []model.BGPHENetworkRow
	for _, tr := range trs[headerIdx+1:] {
		tds := collectTag(tr, "td")
		if len(tds) == 0 {
			continue
		}
		maxCol := colASN
		for _, c := range []int{colName, colV4, colV6} {
			if c > maxCol {
				maxCol = c
			}
		}
		if len(tds) <= maxCol {
			continue
		}

		asnStr, asnNum := parseASNCell(tds[colASN])
		if asnStr == "" || asnNum <= 0 {
			continue
		}
		name := strings.TrimSpace(innerText(tds[colName]))
		if name == "" {
			continue
		}
		rv4, ok1 := parseIntCell(tds[colV4])
		rv6, ok2 := parseIntCell(tds[colV6])
		if !ok1 || !ok2 {
			continue
		}

		out = append(out, model.BGPHENetworkRow{
			ASN:       asnStr,
			ASNNumber: asnNum,
			Name:      name,
			RoutesV4:  rv4,
			RoutesV6:  rv6,
		})
	}

	return out, nil
}

func mapColumns(headers []string) (asn, name, routesV4, routesV6 int) {
	asn, name, routesV4, routesV6 = -1, -1, -1, -1
	for i, h := range headers {
		h = strings.ToLower(strings.TrimSpace(h))
		switch {
		case h == "asn":
			asn = i
		case h == "name":
			name = i
		case strings.Contains(h, "routes") && strings.Contains(h, "v4"):
			routesV4 = i
		case strings.Contains(h, "routes") && strings.Contains(h, "v6"):
			routesV6 = i
		}
	}
	return asn, name, routesV4, routesV6
}

func normalizeHeaderText(n *html.Node) string {
	return strings.TrimSpace(innerText(n))
}

func innerText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
			return
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func parseASNCell(td *html.Node) (display string, num int) {
	// Prefer anchor text / href AS12345
	for _, a := range collectTag(td, "a") {
		href := ""
		for _, attr := range a.Attr {
			if attr.Key == "href" {
				href = strings.TrimSpace(attr.Val)
				break
			}
		}
		raw := strings.TrimSpace(innerText(a))
		if raw != "" {
			display = normalizeASN(raw)
			num = asnNumber(display)
			if num > 0 {
				return display, num
			}
		}
		if href != "" {
			parts := strings.Split(strings.Trim(href, "/"), "/")
			last := parts[len(parts)-1]
			display = normalizeASN(last)
			num = asnNumber(display)
			if num > 0 {
				return display, num
			}
		}
	}
	raw := strings.TrimSpace(innerText(td))
	display = normalizeASN(raw)
	num = asnNumber(display)
	return display, num
}

func normalizeASN(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && (s[0] == 'a' || s[0] == 'A') && (s[1] == 's' || s[1] == 'S') {
		return "AS" + strings.TrimSpace(s[2:])
	}
	if n := asnNumber("AS" + s); n > 0 {
		return "AS" + strings.TrimSpace(s)
	}
	return s
}

func asnNumber(display string) int {
	s := strings.TrimSpace(display)
	s = strings.TrimPrefix(strings.TrimPrefix(s, "AS"), "as")
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func parseIntCell(td *html.Node) (int, bool) {
	raw := strings.TrimSpace(innerText(td))
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return n, true
}
