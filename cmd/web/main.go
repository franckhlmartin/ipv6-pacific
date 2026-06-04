package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pacific-monitor/pacific-monitor/internal/bgphe"
	"github.com/pacific-monitor/pacific-monitor/internal/checks"
	"github.com/pacific-monitor/pacific-monitor/internal/config"
	"github.com/pacific-monitor/pacific-monitor/internal/dotenv"
	"github.com/pacific-monitor/pacific-monitor/internal/httpserver"
	"github.com/pacific-monitor/pacific-monitor/internal/ipv4outage"
	"github.com/pacific-monitor/pacific-monitor/internal/model"
	"github.com/pacific-monitor/pacific-monitor/internal/ogmap"
	"github.com/pacific-monitor/pacific-monitor/internal/probeurls"
	"github.com/pacific-monitor/pacific-monitor/internal/rampscore"
	"github.com/pacific-monitor/pacific-monitor/internal/scoring"
	"github.com/pacific-monitor/pacific-monitor/internal/siteurl"
	"github.com/pacific-monitor/pacific-monitor/internal/summary"
)

//go:embed templates/*.html templates/partials/*.html
var templateFS embed.FS

//go:embed static/css static/js static/img static/favicon.svg static/favicon-16px.ico static/favicon-32px.ico static/favicon-48px.ico static/robots.txt static/well-known/pki-validation/starfield.html
var staticFS embed.FS

func main() {
	dotenv.Load()

	dataDir := getenv("DATA_DIR", "./data")
	addr := getenv("LISTEN", ":8082")
	root := getenv("PROJECT_ROOT", ".")

	pacific, err := config.LoadPacific(root)
	if err != nil {
		log.Printf("warning: load pacific_iso2: %v", err)
	}
	allowed := config.AllowedISO(pacific)

	tmpl, err := template.New("").Funcs(template.FuncMap{
		"rowScore":        scoring.RowScore,
		"dnssecCellClass": scoring.DNSSECCellClass,
		"bgpheRowClass":   bgphe.RowClass,
		"bgpheRowAria":    bgphe.RowAriaLabel,
		"dmarcPctAttr":       dmarcPctAttr,
		"dmarcInspectorURL":  dmarcInspectorURL,
		"rpkiPctAttr":        rpkiPctAttr,
		"rpkiDisplayPct":     rpkiDisplayPct,
		"rpkiStatURL":        rpkiStatURL,
		"rpkiStatusText":     rpkiStatusText,
	}).ParseFS(templateFS, "templates/*.html", "templates/partials/*.html")
	if err != nil {
		log.Fatal(err)
	}
	tmpl566, err := template.New("566.html").ParseFS(templateFS, "templates/566.html", "templates/partials/conn-status.html")
	if err != nil {
		log.Fatal(err)
	}

	probeV4, probeV6, probeDS := probeURLsFromEnv()
	publicSiteURL := strings.TrimSpace(os.Getenv("PUBLIC_SITE_URL"))
	connBundle, err := loadConnStatusBundle(probeV4, probeV6, probeDS, publicSiteURL)
	if err != nil {
		log.Fatalf("conn-status embed bundle: %v", err)
	}
	enrich566 := enrich566Page(connBundle, publicSiteURL)

	outageCfg := ipv4outage.LoadConfig()
	ipv4outage.WarnForceInProduction(outageCfg)

	static, _ := fs.Sub(staticFS, "static")
	wellKnown, _ := fs.Sub(staticFS, "static/well-known")
	rl := httpserver.NewRateLimit(20, 40)
	ogRL := httpserver.NewRateLimit(15, 30)
	mux := http.NewServeMux()

	// Use explicit GET for static assets so Go 1.22+ ServeMux does not conflict with "GET /".
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))
	mux.Handle("GET /.well-known/", http.StripPrefix("/.well-known/", http.FileServer(http.FS(wellKnown))))
	mux.HandleFunc("GET /favicon.ico", serveRootFaviconICO)
	mux.HandleFunc("GET /robots.txt", serveRobotsTxt)
	mux.HandleFunc("GET /sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		serveSitemap(w, r, dataDir, pacific)
	})
	mux.HandleFunc("GET /og/map.png", func(w http.ResponseWriter, r *http.Request) {
		serveOGMapPNG(w, r, dataDir)
	})
	mux.HandleFunc("GET /embed", func(w http.ResponseWriter, r *http.Request) { embedPage(tmpl, w, r) })
	mux.HandleFunc("GET /embed/conn-status", func(w http.ResponseWriter, r *http.Request) {
		serveEmbedConnStatus(tmpl, connBundle, w, r, publicSiteURL)
	})
	mux.HandleFunc("GET /embed/conn-status/details", func(w http.ResponseWriter, r *http.Request) {
		serveEmbedConnStatusDetails(tmpl, connBundle, w, r, publicSiteURL)
	})
	mux.HandleFunc("GET /embed/conn-status.js", func(w http.ResponseWriter, r *http.Request) {
		serveEmbedScript(w, connBundle)
	})
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) { home(tmpl, w, r, dataDir, pacific) })
	mux.HandleFunc("GET /about", func(w http.ResponseWriter, r *http.Request) { aboutPage(tmpl, w, r) })
	mux.HandleFunc("GET /country/{iso}", func(w http.ResponseWriter, r *http.Request) { countryPage(tmpl, w, r, dataDir, pacific, allowed) })
	mux.HandleFunc("GET /api/index.json", func(w http.ResponseWriter, r *http.Request) { serveFile(w, filepath.Join(dataDir, "index.json")) })
	// {iso} must be a full path segment (Go 1.22+); use /api/countries/FJ not .../FJ.json
	mux.HandleFunc("GET /api/countries/{iso}", func(w http.ResponseWriter, r *http.Request) {
		iso := strings.ToUpper(strings.TrimSpace(r.PathValue("iso")))
		if err := config.ValidateISO(allowed, iso); err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		serveFile(w, filepath.Join(dataDir, "countries", iso+".json"))
	})
	mux.HandleFunc("GET /.well-known/security.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("Contact: mailto:security@example.com\nPreferred-Languages: en\n"))
	})
	healthz := func(w http.ResponseWriter, r *http.Request) {
		httpserver.HealthzCORSAllowOrigin(w, r)
		w.Header().Set("Content-Type", "application/json")
		ip, family := httpserver.ClientIPFamily(r)
		payload := struct {
			OK     bool   `json:"ok"`
			IP     string `json:"ip,omitempty"`
			Family string `json:"family"`
		}{OK: true, IP: ip, Family: family}
		_ = json.NewEncoder(w).Encode(payload)
	}
	mux.HandleFunc("GET /api/healthz", healthz)
	mux.HandleFunc("OPTIONS /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		httpserver.HealthzCORSAllowOrigin(w, r)
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/client-ip-family", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		ip, family := httpserver.ClientIPFamily(r)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"family": family,
			"ip":     ip,
		})
	})

	probeConnect := httpserver.ConnectSrcFromProbeEnv()
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := httpserver.RemoteIP(r)
		if strings.HasPrefix(r.URL.Path, "/og/") && !ogRL.Allow(ip) {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api/healthz" && !rl.Allow(ip) {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		mux.ServeHTTP(w, r)
	})
	handler := httpserver.SecurityHeaders(probeConnect,
		ipv4outage.Middleware(outageCfg, tmpl566, enrich566, nil, app))

	certFile := tlsCertPath(root, getenv("TLS_CERT_FILE", "certs/cert.pem"))
	keyFile := tlsCertPath(root, getenv("TLS_KEY_FILE", "certs/key.pem"))
	if _, err := os.Stat(certFile); err != nil {
		log.Fatalf("TLS cert not usable at %s (PROJECT_ROOT=%q): %v", certFile, root, err)
	}
	if _, err := os.Stat(keyFile); err != nil {
		log.Fatalf("TLS key not readable at %s (PROJECT_ROOT=%q): %v", keyFile, root, err)
	}

	if strings.TrimSpace(os.Getenv("PROBE_V4_URL")) != "" && strings.TrimSpace(os.Getenv("PROBE_V6_URL")) != "" {
		log.Print("web: dual-stack border probes configured (PROBE_V4_URL and PROBE_V6_URL set)")
	} else {
		log.Printf("web: dual-stack border probes using defaults for %s (override with PROBE_V4_URL / PROBE_V6_URL in .env)", probeurls.SiteHost())
	}
	if strings.TrimSpace(os.Getenv("PROBE_DS_URL")) != "" {
		log.Print("web: dual-stack preferred probe configured (PROBE_DS_URL set)")
	} else {
		log.Printf("web: PROBE_DS_URL unset — using default https://%s/api/healthz for preferred stack", probeurls.SiteHost())
	}

	if strings.TrimSpace(os.Getenv("PUBLIC_SITE_URL")) == "" {
		log.Print("web: PUBLIC_SITE_URL is unset — canonical and Open Graph URLs use each request's Host; set PUBLIC_SITE_URL when behind a reverse proxy (see .env.example)")
	}

	log.Printf("web listening with TLS on %s (cert %s)", addr, certFile)
	if err := http.ListenAndServeTLS(addr, certFile, keyFile, handler); err != nil {
		log.Fatal(err)
	}
}

func tlsCertPath(root, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(root, p)
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func mergeOutagePageData(data map[string]any) {
	now := time.Now()
	data["PreOutageBanner"] = ipv4outage.InPreOutageWindow(now)
	data["PreOutageDaysUntil"] = ipv4outage.DaysUntilOutage(now)
}

func probeURLsFromEnv() (v4, v6, ds string) {
	cfg := probeurls.Load()
	return cfg.V4, cfg.V6, cfg.DS
}

func serveRootFaviconICO(w http.ResponseWriter, r *http.Request) {
	data, err := staticFS.ReadFile("static/favicon-32px.ico")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/vnd.microsoft.icon")
	_, _ = w.Write(data)
}

func serveRobotsTxt(w http.ResponseWriter, r *http.Request) {
	data, err := staticFS.ReadFile("static/robots.txt")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "sitemap:") {
			continue
		}
		lines = append(lines, line)
	}
	body := strings.Join(lines, "\n")
	if body != "" {
		body += "\n"
	}
	body += "\nSitemap: " + siteurl.AbsoluteURL(r, "/sitemap.xml") + "\n"
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(body))
}

func serveFile(w http.ResponseWriter, path string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	raw, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	w.Write(raw)
}

func seoMerge(r *http.Request, data map[string]any, title, metaDescription string) {
	p := r.URL.Path
	if p == "" {
		p = "/"
	}
	data["MetaDescription"] = metaDescription
	data["CanonicalURL"] = siteurl.AbsoluteURL(r, p)
	data["OgTitle"] = title
	data["OgSiteName"] = "Pacific Islands IPv6 Monitor"
	data["OgImageURL"] = siteurl.AbsoluteURL(r, "/og/map.png")
}

func serveOGMapPNG(w http.ResponseWriter, r *http.Request, dataDir string) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	indexPath := filepath.Join(dataDir, "index.json")
	indexJSON, err := os.ReadFile(indexPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("og map: read index: %v", err)
		}
		indexJSON = []byte("{}")
	}

	svgBytes, err := staticFS.ReadFile("static/img/EEZ_Oceania.svg")
	if err != nil {
		log.Printf("og map: read svg: %v", err)
		writeOGFallback(w)
		return
	}

	type gen struct {
		png  []byte
		etag string
		err  error
	}
	ch := make(chan gen, 1)
	go func() {
		png, etag, err := ogmap.BuildMapPNG(indexJSON, svgBytes)
		ch <- gen{png: png, etag: etag, err: err}
	}()

	select {
	case <-ctx.Done():
		log.Print("og map: generation timed out")
		writeOGFallback(w)
	case out := <-ch:
		if out.err != nil {
			log.Printf("og map: build: %v", out.err)
			writeOGFallback(w)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=300")
		w.Header().Set("ETag", `"`+out.etag+`"`)
		if inm := strings.Trim(r.Header.Get("If-None-Match"), `"`); inm != "" && inm == out.etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		_, _ = w.Write(out.png)
	}
}

func writeOGFallback(w http.ResponseWriter) {
	b, err := ogmap.FallbackPNG()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("ETag", `"og-fallback-v1"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

func home(tmpl *template.Template, w http.ResponseWriter, r *http.Request, dataDir string, pacific *config.PacificList) {
	idxPath := filepath.Join(dataDir, "index.json")
	raw, err := os.ReadFile(idxPath)
	var idx model.Index
	switch {
	case err == nil:
		_ = json.Unmarshal(raw, &idx)
	case os.IsNotExist(err):
		// empty index below
	default:
		log.Printf("home: cannot read %s (check DATA_DIR permissions vs systemd User): %v", idxPath, err)
	}
	gen := "—"
	if !idx.GeneratedAt.IsZero() {
		gen = idx.GeneratedAt.Format(time.RFC3339)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	borderClass := "border--ipv4"
	if httpserver.IsIPv6Client(r) {
		borderClass = "border--ipv6"
	}
	probeV4, probeV6, probeDS := probeURLsFromEnv()
	pageTitle := "Pacific Islands IPv6 Monitor"
	metaDesc := "Pacific Islands IPv6 Monitor — IPv6 deployment estimates for Pacific economies from DNS, mail, and web checks plus APNIC Labs capability data."
	data := map[string]any{
		"Index":         idx,
		"Title":         pageTitle,
		"BorderClass":   borderClass,
		"FooterVariant": "home",
		"Generated":     gen,
		"EEZNotice":     "EEZ overview map: Wikimedia Commons — File:EEZ_Oceania.svg (author STyx, public domain). Source: https://commons.wikimedia.org/wiki/File:EEZ_Oceania.svg",
		"ProbeV4":       probeV4,
		"ProbeV6":       probeV6,
		"ProbeDS":       probeDS,
		"ShowDualProbe": probeV4 != "" && probeV6 != "",
		"Nonce":         httpserver.CSPNonce(r),
	}
	seoMerge(r, data, pageTitle, metaDesc)
	mergeOutagePageData(data)
	_ = tmpl.ExecuteTemplate(w, "home.html", data)
}

func aboutPage(tmpl *template.Template, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	borderClass := "border--ipv4"
	if httpserver.IsIPv6Client(r) {
		borderClass = "border--ipv6"
	}
	probeV4, probeV6, probeDS := probeURLsFromEnv()
	pageTitle := "About — Pacific Islands IPv6 Monitor"
	metaDesc := "Pacific Islands IPv6 Council and this deployment monitor — IPv6 roadmap for the Pacific, measurements, and council leadership."
	data := map[string]any{
		"Title":         pageTitle,
		"BorderClass":   borderClass,
		"FooterVariant": "about",
		"ProbeV4":       probeV4,
		"ProbeV6":       probeV6,
		"ProbeDS":       probeDS,
		"ShowDualProbe": probeV4 != "" && probeV6 != "",
		"Nonce":         httpserver.CSPNonce(r),
	}
	seoMerge(r, data, pageTitle, metaDesc)
	mergeOutagePageData(data)
	_ = tmpl.ExecuteTemplate(w, "about.html", data)
}

func countryPage(tmpl *template.Template, w http.ResponseWriter, r *http.Request, dataDir string, pacific *config.PacificList, allowed map[string]struct{}) {
	iso := strings.ToUpper(strings.TrimSpace(r.PathValue("iso")))
	if err := config.ValidateISO(allowed, iso); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	path := filepath.Join(dataDir, "countries", iso+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		countryComingSoon(tmpl, w, r, iso, pacific)
		return
	}
	var cf model.CountryFile
	if err := json.Unmarshal(raw, &cf); err != nil {
		http.Error(w, "invalid data", http.StatusInternalServerError)
		return
	}
	name := cf.Name
	if pacific != nil {
		for _, c := range pacific.Countries {
			if strings.ToUpper(c.ISO2) == iso {
				name = c.Name
				break
			}
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	borderClass := "border--ipv4"
	if httpserver.IsIPv6Client(r) {
		borderClass = "border--ipv6"
	}
	probeV4, probeV6, probeDS := probeURLsFromEnv()
	sum := summary.FromDomains(cf.Domains)
	economyScorePct := scoring.EconomyDeploymentScorePct(cf.Domains)
	checkLegend := checks.CountryLegendCheckExplanations()
	statusLegend := checks.CountryLegendStatusItems()
	locationLegend := checks.CountryLegendLocationItems()
	scoreLegend := scoring.CountryScoreLegend()
	hasBGPHETable := cf.BGPHurricaneElectric != nil && len(cf.BGPHurricaneElectric.Networks) > 0
	log.Printf("country legend rendered iso=%s status_items=%d check_items=%d location_items=%d has_results=%t has_bgp_he=%t", iso, len(statusLegend), len(checkLegend), len(locationLegend), len(cf.Domains) > 0, hasBGPHETable)
	pageTitle := name + " — Pacific Islands IPv6 Monitor"
	metaDesc := fmt.Sprintf("%s — IPv6 deployment estimates from DNS, mail, web checks and APNIC Labs data (Pacific Islands IPv6 Monitor).", name)
	data := map[string]any{
		"ISO":             iso,
		"Name":            name,
		"Country":         cf,
		"Summary":         sum,
		"Title":           pageTitle,
		"BorderClass":     borderClass,
		"FooterVariant":   "country",
		"ProbeV4":         probeV4,
		"ProbeV6":         probeV6,
		"ProbeDS":         probeDS,
		"ShowDualProbe":   probeV4 != "" && probeV6 != "",
		"HasResults":      len(cf.Domains) > 0,
		"HasBGPHETable":   hasBGPHETable,
		"EconomyScorePct": economyScorePct,
		"Generated":       cf.GeneratedAt.Format(time.RFC3339),
		"LegendStatus":    statusLegend,
		"LegendChecks":    checkLegend,
		"LegendLocation":  locationLegend,
		"ScoreLegend":     scoreLegend,
		"DMARCRampLegend": scoring.DMARCRampLegend(),
		"RPKIRampLegend":  scoring.RPKIRampLegend(),
		"Nonce":           httpserver.CSPNonce(r),
	}
	seoMerge(r, data, pageTitle, metaDesc)
	_ = tmpl.ExecuteTemplate(w, "country.html", data)
}

func countryComingSoon(tmpl *template.Template, w http.ResponseWriter, r *http.Request, iso string, pacific *config.PacificList) {
	name := iso
	if pacific != nil {
		for _, c := range pacific.Countries {
			if strings.ToUpper(c.ISO2) == iso {
				name = c.Name
				break
			}
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	borderClass := "border--ipv4"
	if httpserver.IsIPv6Client(r) {
		borderClass = "border--ipv6"
	}
	probeV4, probeV6, probeDS := probeURLsFromEnv()
	pageTitle := name + " — Pacific Islands IPv6 Monitor"
	metaDesc := fmt.Sprintf("%s — Pacific Islands IPv6 Monitor; collector results for this economy are not yet published.", name)
	data := map[string]any{
		"ISO":           iso,
		"Name":          name,
		"Title":         pageTitle,
		"BorderClass":   borderClass,
		"FooterVariant": "country",
		"ProbeV4":       probeV4,
		"ProbeV6":       probeV6,
		"ProbeDS":       probeDS,
		"ShowDualProbe": probeV4 != "" && probeV6 != "",
		"Nonce":         httpserver.CSPNonce(r),
	}
	seoMerge(r, data, pageTitle, metaDesc)
	_ = tmpl.ExecuteTemplate(w, "country_soon.html", data)
}

// dmarcPctAttr returns data-pct for ramp coloring (empty = grey).
func dmarcPctAttr(col model.DMARCColumn) string {
	switch col.State {
	case "absent":
		return "0"
	case "present":
		return fmt.Sprintf("%.6f", col.ScorePct)
	default:
		return ""
	}
}

// rpkiDisplayPct returns the RPKI ramp percentage from sampled counts (not stored JSON).
func rpkiDisplayPct(row model.BGPHENetworkRow) float64 {
	pct, ok := rampscore.RPKIScorePct(row.RPKIValid, row.RPKIInvalid, row.RPKIUnknown, row.RPKICheckedPrefixes, row.RPKIError)
	if !ok {
		return 0
	}
	return pct
}

// rpkiPctAttr returns data-pct when sampled RPKI data exists.
func rpkiPctAttr(row model.BGPHENetworkRow) string {
	if row.RPKIError != "" || row.RPKICheckedPrefixes <= 0 {
		return ""
	}
	return fmt.Sprintf("%.6f", rpkiDisplayPct(row))
}

// dmarcInspectorURL links to dmarcian DMARC Inspector for the apex domain.
func dmarcInspectorURL(domain string) string {
	domain = strings.TrimSpace(strings.ToLower(domain))
	if domain == "" {
		return "https://dmarcian.com/dmarc-inspector/"
	}
	v := url.Values{}
	v.Set("domain", domain)
	return "https://dmarcian.com/dmarc-inspector/?" + v.Encode()
}

// rpkiStatURL links to the RIPEstat resource overview for this ASN.
func rpkiStatURL(row model.BGPHENetworkRow) string {
	asn := strings.TrimSpace(row.ASN)
	if asn == "" && row.ASNNumber > 0 {
		asn = fmt.Sprintf("AS%d", row.ASNNumber)
	}
	if !strings.HasPrefix(strings.ToUpper(asn), "AS") && row.ASNNumber > 0 {
		asn = fmt.Sprintf("AS%d", row.ASNNumber)
	}
	if asn == "" {
		return "https://stat.ripe.net/"
	}
	return "https://stat.ripe.net/resource/" + url.PathEscape(asn) + "#tab=overview"
}

// rpkiStatusText summarizes RPKI sampling for the status column.
func rpkiStatusText(row model.BGPHENetworkRow) string {
	if row.RPKIError != "" {
		return row.RPKIError
	}
	if row.RPKICheckedPrefixes <= 0 {
		return "—"
	}
	if row.RPKIWorstStatus != "" {
		return fmt.Sprintf("%s (%d/%d valid)", row.RPKIWorstStatus, row.RPKIValid, row.RPKICheckedPrefixes)
	}
	return fmt.Sprintf("%d/%d valid", row.RPKIValid, row.RPKICheckedPrefixes)
}
