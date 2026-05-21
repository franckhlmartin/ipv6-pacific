package collector

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pacific-monitor/pacific-monitor/internal/model"
	"github.com/pacific-monitor/pacific-monitor/internal/ripestat"
)

// EnrichBGPRPKI updates RPKI fields on BGP rows using RIPEstat, with carry-forward and refresh interval.
func EnrichBGPRPKI(ctx context.Context, he *model.BGPHETable, prev *model.BGPHETable, client *ripestat.Client, verbose bool, iso2 string) {
	if he == nil || len(he.Networks) == 0 || client == nil {
		return
	}
	prevBy := map[int]model.BGPHENetworkRow{}
	if prev != nil {
		for _, row := range prev.Networks {
			if row.ASNNumber > 0 {
				prevBy[row.ASNNumber] = row
			}
		}
	}
	maxP := intEnv("COLLECTOR_RPKI_MAX_PREFIXES_PER_ASN", 10)
	refresh := durationEnv("COLLECTOR_RPKI_REFRESH_INTERVAL", 168*time.Hour)
	now := time.Now().UTC()

	for i := range he.Networks {
		row := &he.Networks[i]
		if row.ASNNumber <= 0 {
			continue
		}
		if !row.RPKICheckedAt.IsZero() && now.Sub(row.RPKICheckedAt) < refresh {
			continue
		}
		if old, ok := prevBy[row.ASNNumber]; ok && !old.RPKICheckedAt.IsZero() && now.Sub(old.RPKICheckedAt) < refresh {
			copyRPKI(row, &old)
			continue
		}
		if verbose {
			log.Printf("[collector] %s: RPKI check %s (AS%d)", iso2, row.ASN, row.ASNNumber)
		}
		client.EnrichASN(ctx, row, maxP)
		if row.RPKIError != "" && verbose {
			log.Printf("[collector] %s: RPKI %s: %s", iso2, row.ASN, row.RPKIError)
		}
	}
}

func copyRPKI(dst, src *model.BGPHENetworkRow) {
	dst.RPKICheckedPrefixes = src.RPKICheckedPrefixes
	dst.RPKIValid = src.RPKIValid
	dst.RPKIInvalid = src.RPKIInvalid
	dst.RPKIUnknown = src.RPKIUnknown
	dst.RPKIScorePct = src.RPKIScorePct
	dst.RPKIWorstStatus = src.RPKIWorstStatus
	dst.RPKIError = src.RPKIError
	dst.RPKISourceURL = src.RPKISourceURL
	dst.RPKICheckedAt = src.RPKICheckedAt
}

func intEnv(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func durationEnv(key string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func SkipRPKI() bool {
	return strings.TrimSpace(os.Getenv("COLLECTOR_SKIP_RPKI")) == "1"
}

// NewRipestatClient builds a RIPEstat client from env and the shared HTTP client.
func NewRipestatClient(hc *http.Client) *ripestat.Client {
	sourceApp := strings.TrimSpace(os.Getenv("RIPESTAT_SOURCEAPP"))
	if sourceApp == "" {
		sourceApp = "pacific-ipv6-monitor"
	}
	interval := durationEnv("COLLECTOR_RPKI_MIN_INTERVAL", 300*time.Millisecond)
	return ripestat.NewClient(hc, sourceApp, interval)
}
