package ogmap

import "strings"

// titleToISO mirrors cmd/web/static/js/map-home.js TITLE_TO_ISO (label text → ISO2).
var titleToISO = map[string]string{
	"American Samoa (US)":            "AS",
	"Cook Islands (NZ)":              "CK",
	"Federated States of Micronesia": "FM",
	"Fiji":                           "FJ",
	"French Polynesia (Fr)":          "PF",
	"Kiribati (Gilbert Islands)":     "KI",
	"Line Islands (Kiribati)":        "KI",
	"Marshalls":                      "MH",
	"Nauru":                          "NR",
	"New Caledonia":                  "NC",
	"Niue (NZ)":                      "NU",
	"Northern Marianas (US)":         "MP",
	"Guam (US)":                      "GU",
	"Papua New Guinea":               "PG",
	"Palau":                          "PW",
	"Phoenix Islands (Kiribati)":     "KI",
	"Samoa":                          "WS",
	"Solomon Islands":                "SB",
	"Tokelau (NZ)":                   "TK",
	"Tonga":                          "TO",
	"Tuvalu":                         "TV",
	"Vanuatu":                        "VU",
	"Wallis and Futuna (Fr)":         "WF",
}

// ISOForTerritoryTitle returns the monitored ISO2 for an EEZ path <title> text, or "".
func ISOForTerritoryTitle(title string) string {
	key := normalizeTitle(title)
	if key == "" {
		return ""
	}
	return titleToISO[key]
}

func normalizeTitle(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}
