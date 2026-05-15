package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pacific-monitor/pacific-monitor/internal/apniclabs"
	"github.com/pacific-monitor/pacific-monitor/internal/bgphe"
	"github.com/pacific-monitor/pacific-monitor/internal/checks"
	"github.com/pacific-monitor/pacific-monitor/internal/config"
	"github.com/pacific-monitor/pacific-monitor/internal/dotenv"
	"github.com/pacific-monitor/pacific-monitor/internal/indexbuilder"
	"github.com/pacific-monitor/pacific-monitor/internal/model"
	"github.com/pacific-monitor/pacific-monitor/internal/storage"
)

func main() {
	dotenv.Load()

	runOnce := flag.Bool("run-once", false, "collect then exit: all countries, or only --country if set")
	verbose := flag.Bool("verbose", false, "per-domain progress logs (defaults on when using -run-once)")
	root := flag.String("C", ".", "project root (contains config/ and data/)")
	countryFlag := flag.String("country", "", "ISO2 code (e.g. FJ): with daemon, run this country first then continue round-robin; with -run-once, collect only this country")
	flag.Parse()
	if *runOnce {
		*verbose = true
	}

	dataDir := getenv("DATA_DIR", filepath.Join(*root, "data"))
	if err := os.MkdirAll(filepath.Join(dataDir, "countries"), 0o755); err != nil {
		log.Fatal(err)
	}

	pacific, err := config.LoadPacific(*root)
	if err != nil {
		log.Fatalf("load pacific_iso2: %v", err)
	}

	chk := checksFromEnv()
	httpClient := &http.Client{Timeout: 30 * time.Second}
	ctx := context.Background()

	if *runOnce {
		if err := runOncePass(ctx, *root, dataDir, pacific, chk, httpClient, strings.TrimSpace(strings.ToUpper(*countryFlag)), *verbose); err != nil {
			log.Fatal(err)
		}
		return
	}

	interval := durationEnv("COLLECTOR_PER_COUNTRY_INTERVAL", 10*time.Minute)
	t := time.NewTicker(interval)
	defer t.Stop()

	startIdx, err := startingIndex(pacific, strings.TrimSpace(strings.ToUpper(*countryFlag)))
	if err != nil {
		log.Fatal(err)
	}
	c0 := pacific.Countries[startIdx]
	if *countryFlag != "" {
		log.Printf("collector daemon: first collection %s (%s), then every %v rotating", c0.ISO2, c0.Name, interval)
	} else {
		log.Printf("collector daemon: first collection %s (%s) immediately, then every %v", c0.ISO2, c0.Name, interval)
	}

	if err := collectCountry(ctx, *root, dataDir, pacific, c0, chk, httpClient, *verbose); err != nil {
		log.Printf("[collector] country %s: %v", c0.ISO2, err)
	}

	next := (startIdx + 1) % len(pacific.Countries)
	for {
		<-t.C
		c := pacific.Countries[next%len(pacific.Countries)]
		next++
		if err := collectCountry(ctx, *root, dataDir, pacific, c, chk, httpClient, *verbose); err != nil {
			log.Printf("[collector] country %s: %v", c.ISO2, err)
		}
	}
}

// startingIndex returns index of --country or 0; validates ISO is in the Pacific list.
func startingIndex(pacific *config.PacificList, iso string) (int, error) {
	if iso == "" {
		return 0, nil
	}
	for i, c := range pacific.Countries {
		if strings.EqualFold(c.ISO2, iso) {
			return i, nil
		}
	}
	return 0, fmt.Errorf("unknown country %q (not in config/pacific_iso2.yaml)", iso)
}

func runOncePass(ctx context.Context, root, dataDir string, pacific *config.PacificList, chk checks.Config, hc *http.Client, iso string, verbose bool) error {
	if iso != "" {
		log.Printf("[collector] run-once: single country %s", iso)
		c, err := countryByISO(pacific, iso)
		if err != nil {
			return err
		}
		if err := collectCountry(ctx, root, dataDir, pacific, c, chk, hc, verbose); err != nil {
			log.Printf("[collector] country %s: %v", c.ISO2, err)
		}
		return nil
	}
	log.Printf("[collector] run-once: all %d economies from config/pacific_iso2.yaml", len(pacific.Countries))
	return runPass(ctx, root, dataDir, pacific, chk, hc, verbose)
}

func countryByISO(pacific *config.PacificList, iso string) (config.PacificCountry, error) {
	var zero config.PacificCountry
	for _, c := range pacific.Countries {
		if strings.EqualFold(c.ISO2, iso) {
			return c, nil
		}
	}
	return zero, fmt.Errorf("unknown country %q (not in config/pacific_iso2.yaml)", iso)
}

func runPass(ctx context.Context, root, dataDir string, pacific *config.PacificList, chk checks.Config, hc *http.Client, verbose bool) error {
	for i, c := range pacific.Countries {
		log.Printf("[collector] run-once: [%d/%d] starting %s (%s)", i+1, len(pacific.Countries), c.ISO2, c.Name)
		if err := collectCountry(ctx, root, dataDir, pacific, c, chk, hc, verbose); err != nil {
			log.Printf("[collector] country %s: %v", c.ISO2, err)
		}
	}
	log.Printf("[collector] run-once: finished all %d economies", len(pacific.Countries))
	return nil
}

func finishIndex(dataDir string, pacific *config.PacificList) error {
	idxPath := filepath.Join(dataDir, "index.json")
	if err := indexbuilder.Rebuild(dataDir, pacific); err != nil {
		log.Printf("[collector] rebuild index: %v", err)
		return err
	}
	log.Printf("[collector] index written %s", idxPath)
	applyDataOwnership(dataDir)
	return nil
}

func collectCountry(ctx context.Context, root, dataDir string, pacific *config.PacificList, c config.PacificCountry, chk checks.Config, hc *http.Client, verbose bool) error {
	path := filepath.Join(root, "config", "domains", strings.ToUpper(c.ISO2)+".yaml")
	df, err := config.LoadDomainsFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if verbose {
				log.Printf("[collector] skip %s: no %s", c.ISO2, path)
			}
			return nil
		}
		return err
	}

	out := filepath.Join(dataDir, "countries", strings.ToUpper(c.ISO2)+".json")
	// Build into *.json.partial; rename to *.json only after the full pass (domains + APNIC + HE BGP)
	// so the live file — and the public site — never flip to an in-progress snapshot.
	stagingPath := out + ".partial"

	var prevAPNIC *model.APNICSnapshot
	var prevBGPHE *model.BGPHETable
	var prevDomains []model.DomainResult
	if raw, err := os.ReadFile(out); err == nil {
		var old model.CountryFile
		if json.Unmarshal(raw, &old) == nil {
			prevAPNIC = old.APNICLabs
			prevBGPHE = old.BGPHurricaneElectric
			prevDomains = old.Domains
		}
	}

	writeCountry := func(results []model.DomainResult, ap *model.APNICSnapshot, he *model.BGPHETable) error {
		cf := model.CountryFile{
			ISO2:                 strings.ToUpper(c.ISO2),
			Name:                 c.Name,
			GeneratedAt:          time.Now().UTC(),
			CollectorVersion:     model.CollectorVersion,
			Domains:              results,
			APNICLabs:            ap,
			BGPHurricaneElectric: he,
		}
		return storage.WriteJSON(stagingPath, cf)
	}

	nDom := len(df.Domains)
	log.Printf("[collector] %s (%s): measuring %d domain(s) from %s", c.ISO2, c.Name, nDom, path)
	if nDom == 0 {
		log.Printf("[collector] %s: domain list is empty — nothing to measure", c.ISO2)
	}

	apnicCarry := (*model.APNICSnapshot)(nil)
	if !c.ExcludeAPNIC {
		apnicCarry = prevAPNIC
	}

	results := seedDomainResults(df, prevDomains)
	if nDom > 0 {
		if err := writeCountry(results, apnicCarry, prevBGPHE); err != nil {
			return err
		}
	}

	for i, entry := range df.Domains {
		if verbose {
			log.Printf("[collector] %s domain %d/%d: start %s", c.ISO2, i+1, nDom, entry.Domain)
			log.Printf("[collector] %s | budget | DomainDeadline=%s (entire domain: DNS+Mail+Web+DNSSEC)", entry.Domain, chk.DomainDeadline)
		}
		t0 := time.Now()
		dctx, cancel := context.WithTimeout(ctx, chk.DomainDeadline)
		dchk := chk
		if verbose {
			dom := entry.Domain
			dchk.LogStep = func(phase, timeoutDesc, summary string) {
				log.Printf("[collector] %s | %-8s | %s | %s", dom, phase, timeoutDesc, summary)
			}
		}
		res := checks.RunDomain(dctx, entry.Domain, dchk, checks.DomainMeta{
			Organization: entry.Organization,
			Sector:       entry.Sector,
			WebURL:       entry.WebURL,
		})
		cancel()
		res.CollectedAt = time.Now().UTC()
		elapsed := time.Since(t0).Round(time.Millisecond)
		results[i] = res
		if verbose {
			errMsg := res.Error
			if errMsg == "" {
				errMsg = "-"
			}
			log.Printf("[collector] %s domain %s: done in %s rollup=%s dns=%s mail=%s web=%s err=%s",
				c.ISO2, entry.Domain, elapsed, res.RollupClass, res.DNS.Class, res.Mail.Class, res.Web.Class, errMsg)
		}
		if err := writeCountry(results, apnicCarry, prevBGPHE); err != nil {
			return err
		}
		if verbose {
			log.Printf("[collector] %s: updated staging %s (%d/%d domains, collected_at=%s)", c.ISO2, stagingPath, i+1, nDom, res.CollectedAt.Format(time.RFC3339Nano))
		}
	}

	var ap *model.APNICSnapshot
	if !c.ExcludeAPNIC {
		log.Printf("[collector] %s: fetching APNIC Labs v6economy/%s.json …", c.ISO2, strings.ToUpper(c.ISO2))
		actx, cancel := context.WithTimeout(ctx, 30*time.Second)
		snap, err := apniclabs.FetchLatest(actx, c.ISO2, hc)
		cancel()
		if err != nil {
			log.Printf("[collector] %s: APNIC fetch failed: %v", c.ISO2, err)
			ap = prevAPNIC
		} else {
			ap = snap
			log.Printf("[collector] %s: APNIC IPv6 preferred ~%.2f%% (date %s)", c.ISO2, snap.PreferredPct, snap.Date)
		}
	} else {
		log.Printf("[collector] %s: skipping APNIC (exclude_apnic)", c.ISO2)
		ap = nil
	}

	var heTable *model.BGPHETable
	if skipHEBGP() {
		log.Printf("[collector] %s: skipping Hurricane Electric BGP (COLLECTOR_SKIP_HE_BGP)", c.ISO2)
		heTable = prevBGPHE
	} else {
		log.Printf("[collector] %s: fetching Hurricane Electric bgp.he.net/country/%s …", c.ISO2, strings.ToUpper(c.ISO2))
		hctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		heSnap, err := bgphe.FetchCountryNetworks(hctx, c.ISO2, hc)
		cancel()
		if err != nil {
			log.Printf("[collector] %s: Hurricane Electric BGP fetch failed: %v", c.ISO2, err)
			heTable = prevBGPHE
		} else {
			heTable = heSnap
			log.Printf("[collector] %s: Hurricane Electric BGP networks=%d (fetched_at=%s)", c.ISO2, len(heSnap.Networks), heSnap.FetchedAt.Format(time.RFC3339))
		}
	}

	if err := writeCountry(results, ap, heTable); err != nil {
		return err
	}
	if err := os.Rename(stagingPath, out); err != nil {
		return fmt.Errorf("publish %s: %w", out, err)
	}
	log.Printf("[collector] %s: published %s (%d domain row(s))", c.ISO2, out, len(results))
	return finishIndex(dataDir, pacific)
}

// seedDomainResults builds one slot per YAML domain: reuse the last JSON row when the name
// still appears in config (so incremental writes keep unmeasured rows); new names get an
// empty shell. Domains removed from YAML are omitted entirely.
func seedDomainResults(df *config.DomainsFile, prevDomains []model.DomainResult) []model.DomainResult {
	prevBy := make(map[string]model.DomainResult, len(prevDomains))
	for _, d := range prevDomains {
		k := strings.ToLower(strings.TrimSpace(d.Domain))
		if k == "" {
			continue
		}
		prevBy[k] = d
	}
	out := make([]model.DomainResult, len(df.Domains))
	for i, entry := range df.Domains {
		k := strings.ToLower(entry.Domain)
		if old, ok := prevBy[k]; ok {
			old.Domain = entry.Domain
			old.Organization = entry.Organization
			old.Sector = entry.Sector
			out[i] = old
			continue
		}
		out[i] = model.DomainResult{
			Domain:       entry.Domain,
			Organization: entry.Organization,
			Sector:       entry.Sector,
		}
	}
	return out
}

func checksFromEnv() checks.Config {
	c := checks.DefaultConfig()
	if v := getenv("DNS_TIMEOUT", ""); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.DNSResolveTimeout = d
		}
	}
	if v := getenv("HTTP_CLIENT_TIMEOUT", ""); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.HTTPTimeout = d
		}
	}
	if v := getenv("SMTP_TIMEOUT", ""); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.SMTPTimeout = d
		}
	}
	if v := getenv("CHECK_DOMAIN_DEADLINE", ""); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.DomainDeadline = d
		}
	}
	return c
}

func durationEnv(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// skipHEBGP skips bgp.he.net scraping when COLLECTOR_SKIP_HE_BGP=1 (ops kill switch).
func skipHEBGP() bool {
	return strings.TrimSpace(os.Getenv("COLLECTOR_SKIP_HE_BGP")) == "1"
}
