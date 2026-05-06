package indexbuilder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pacific-monitor/pacific-monitor/internal/config"
	"github.com/pacific-monitor/pacific-monitor/internal/model"
	"github.com/pacific-monitor/pacific-monitor/internal/scoring"
	"github.com/pacific-monitor/pacific-monitor/internal/storage"
	"github.com/pacific-monitor/pacific-monitor/internal/summary"
)

// Rebuild scans data/countries/*.json and writes data/index.json.
func Rebuild(dataDir string, pacific *config.PacificList) error {
	meta := map[string]config.PacificCountry{}
	for _, c := range pacific.Countries {
		meta[strings.ToUpper(c.ISO2)] = c
	}

	dir := filepath.Join(dataDir, "countries")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		idx := model.Index{
			GeneratedAt:      time.Now().UTC(),
			CollectorVersion: model.CollectorVersion,
			Countries:        []model.IndexCountry{},
		}
		return storage.WriteJSON(filepath.Join(dataDir, "index.json"), idx)
	}
	if err != nil {
		return err
	}
	var list []model.IndexCountry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var cf model.CountryFile
		if err := json.Unmarshal(raw, &cf); err != nil {
			continue
		}
		iso := strings.ToUpper(strings.TrimSuffix(e.Name(), ".json"))
		ic := model.IndexCountry{
			ISO2:               iso,
			Name:               cf.Name,
			DomainCount:        len(cf.Domains),
			Summary:            summary.FromDomains(cf.Domains),
			DeploymentScorePct: scoring.EconomyDeploymentScorePct(cf.Domains),
			APNICLabs:          cf.APNICLabs,
		}
		if m, ok := meta[iso]; ok {
			if m.Name != "" {
				ic.Name = m.Name
			}
			if len(m.Centroid) == 2 {
				ic.Centroid = m.Centroid
			}
		}
		t := cf.GeneratedAt
		ic.LastCollected = &t
		list = append(list, ic)
	}
	idx := model.Index{
		GeneratedAt:      time.Now().UTC(),
		CollectorVersion: model.CollectorVersion,
		Countries:        list,
	}
	path := filepath.Join(dataDir, "index.json")
	return storage.WriteJSON(path, idx)
}
