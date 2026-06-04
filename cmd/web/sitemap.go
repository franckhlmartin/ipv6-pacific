package main

import (
	"encoding/json"
	"encoding/xml"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pacific-monitor/pacific-monitor/internal/config"
	"github.com/pacific-monitor/pacific-monitor/internal/model"
	"github.com/pacific-monitor/pacific-monitor/internal/siteurl"
)

type sitemapURLSet struct {
	XMLName xml.Name          `xml:"urlset"`
	Xmlns   string            `xml:"xmlns,attr"`
	URLs    []sitemapURLEntry `xml:"url"`
}

type sitemapURLEntry struct {
	Loc     string `xml:"loc"`
	Lastmod string `xml:"lastmod,omitempty"`
}

func serveSitemap(w http.ResponseWriter, r *http.Request, dataDir string, pacific *config.PacificList) {
	type pathMod struct {
		path   string
		modUTC time.Time
		hasMod bool
	}
	var rows []pathMod

	idxPath := filepath.Join(dataDir, "index.json")
	var homeMod time.Time
	var homeHas bool
	if raw, err := os.ReadFile(idxPath); err == nil {
		var idx model.Index
		if err := json.Unmarshal(raw, &idx); err == nil && !idx.GeneratedAt.IsZero() {
			homeMod = idx.GeneratedAt.UTC()
			homeHas = true
		}
	}
	rows = append(rows, pathMod{path: "/", modUTC: homeMod, hasMod: homeHas})
	rows = append(rows, pathMod{path: "/about"})
	rows = append(rows, pathMod{path: "/embed"})

	var isos []string
	if pacific != nil {
		for _, c := range pacific.Countries {
			u := strings.ToUpper(strings.TrimSpace(c.ISO2))
			if len(u) == 2 {
				isos = append(isos, u)
			}
		}
	}
	sort.Strings(isos)
	for _, iso := range isos {
		p := filepath.Join(dataDir, "countries", iso+".json")
		st, err := os.Stat(p)
		row := pathMod{path: "/country/" + iso}
		if err == nil {
			row.modUTC = st.ModTime().UTC()
			row.hasMod = true
		}
		rows = append(rows, row)
	}

	out := sitemapURLSet{
		XMLName: xml.Name{Local: "urlset"},
		Xmlns:   "http://www.sitemaps.org/schemas/sitemap/0.9",
	}
	for _, row := range rows {
		e := sitemapURLEntry{Loc: siteurl.AbsoluteURL(r, row.path)}
		if row.hasMod {
			e.Lastmod = row.modUTC.Format(time.RFC3339)
		}
		out.URLs = append(out.URLs, e)
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	if _, err := w.Write([]byte(xml.Header)); err != nil {
		return
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(out); err != nil {
		log.Printf("sitemap: encode: %v", err)
	}
}
