package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pacific-monitor/pacific-monitor/internal/model"
)

func TestWriteJSONRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "index.json")
	idx := model.Index{
		CollectorVersion: model.CollectorVersion,
		Countries: []model.IndexCountry{
			{ISO2: "FJ", Name: "Fiji", DomainCount: 1},
		},
	}
	if err := WriteJSON(path, idx); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 20 {
		t.Fatalf("unexpected short file: %s", data)
	}
}
