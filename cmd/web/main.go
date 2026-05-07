package main

import (
	"embed"
	"encoding/json"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/pacific-monitor/pacific-monitor/internal/checks"
	"github.com/pacific-monitor/pacific-monitor/internal/config"
	"github.com/pacific-monitor/pacific-monitor/internal/httpserver"
	"github.com/pacific-monitor/pacific-monitor/internal/model"
	"github.com/pacific-monitor/pacific-monitor/internal/scoring"
	"github.com/pacific-monitor/pacific-monitor/internal/summary"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/css static/js static/img
var staticFS embed.FS

func main() {
	_ = godotenv.Load(".env")
	_ = godotenv.Load(".env.local")

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
	}).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		log.Fatal(err)
	}

	static, _ := fs.Sub(staticFS, "static")
	rl := httpserver.NewRateLimit(20, 40)
	mux := http.NewServeMux()

	// Use explicit GET for static assets so Go 1.22+ ServeMux does not conflict with "GET /".
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) { home(tmpl, w, r, dataDir, pacific) })
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
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("GET /api/client-ip-family", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		v := "ipv4"
		if httpserver.IsIPv6Client(r) {
			v = "ipv6"
		}
		json.NewEncoder(w).Encode(map[string]string{"family": v})
	})

	handler := httpserver.SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api/healthz" && !rl.Allow(httpserver.RemoteIP(r)) {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		mux.ServeHTTP(w, r)
	}))

	certFile := tlsCertPath(root, getenv("TLS_CERT_FILE", "certs/cert.pem"))
	keyFile := tlsCertPath(root, getenv("TLS_KEY_FILE", "certs/key.pem"))
	if _, err := os.Stat(certFile); err != nil {
		log.Fatalf("TLS cert not usable at %s (PROJECT_ROOT=%q): %v", certFile, root, err)
	}
	if _, err := os.Stat(keyFile); err != nil {
		log.Fatalf("TLS key not readable at %s (PROJECT_ROOT=%q): %v", keyFile, root, err)
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

func serveFile(w http.ResponseWriter, path string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	raw, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	w.Write(raw)
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
	probeV4 := os.Getenv("PROBE_V4_URL")
	probeV6 := os.Getenv("PROBE_V6_URL")
	_ = tmpl.ExecuteTemplate(w, "home.html", map[string]any{
		"Index":         idx,
		"Title":         "Pacific Islands IPv6 Monitor",
		"BorderClass":   borderClass,
		"Generated":     gen,
		"EEZNotice":     "EEZ overview map: Wikimedia Commons — File:EEZ_Oceania.svg (author STyx, public domain). Source: https://commons.wikimedia.org/wiki/File:EEZ_Oceania.svg",
		"ProbeV4":       probeV4,
		"ProbeV6":       probeV6,
		"ShowDualProbe": probeV4 != "" && probeV6 != "",
		"Nonce":         httpserver.CSPNonce(r),
	})
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
	probeV4 := os.Getenv("PROBE_V4_URL")
	probeV6 := os.Getenv("PROBE_V6_URL")
	sum := summary.FromDomains(cf.Domains)
	economyScorePct := scoring.EconomyDeploymentScorePct(cf.Domains)
	checkLegend := checks.CountryLegendCheckExplanations()
	statusLegend := checks.CountryLegendStatusItems()
	locationLegend := checks.CountryLegendLocationItems()
	scoreLegend := scoring.CountryScoreLegend()
	log.Printf("country legend rendered iso=%s status_items=%d check_items=%d location_items=%d has_results=%t", iso, len(statusLegend), len(checkLegend), len(locationLegend), len(cf.Domains) > 0)
	_ = tmpl.ExecuteTemplate(w, "country.html", map[string]any{
		"ISO":             iso,
		"Name":            name,
		"Country":         cf,
		"Summary":         sum,
		"Title":           name + " — Pacific Islands IPv6 Monitor",
		"BorderClass":     borderClass,
		"ProbeV4":         probeV4,
		"ProbeV6":         probeV6,
		"ShowDualProbe":   probeV4 != "" && probeV6 != "",
		"HasResults":      len(cf.Domains) > 0,
		"EconomyScorePct": economyScorePct,
		"Generated":       cf.GeneratedAt.Format(time.RFC3339),
		"LegendStatus":    statusLegend,
		"LegendChecks":    checkLegend,
		"LegendLocation":  locationLegend,
		"ScoreLegend":     scoreLegend,
		"Nonce":           httpserver.CSPNonce(r),
	})
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
	probeV4 := os.Getenv("PROBE_V4_URL")
	probeV6 := os.Getenv("PROBE_V6_URL")
	_ = tmpl.ExecuteTemplate(w, "country_soon.html", map[string]any{
		"ISO":           iso,
		"Name":          name,
		"Title":         name + " — Pacific Islands IPv6 Monitor",
		"BorderClass":   borderClass,
		"ProbeV4":       probeV4,
		"ProbeV6":       probeV6,
		"ShowDualProbe": probeV4 != "" && probeV6 != "",
		"Nonce":         httpserver.CSPNonce(r),
	})
}
