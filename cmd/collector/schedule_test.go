package main

import (
	"testing"

	"github.com/pacific-monitor/pacific-monitor/internal/config"
)

func TestCountrySchedule_pinsFirstAndKeepsAll(t *testing.T) {
	countries := []config.PacificCountry{
		{ISO2: "FJ", Name: "Fiji"},
		{ISO2: "PG", Name: "Papua New Guinea"},
		{ISO2: "WS", Name: "Samoa"},
	}
	schedule, err := countrySchedule(countries, "PG")
	if err != nil {
		t.Fatal(err)
	}
	if len(schedule) != len(countries) {
		t.Fatalf("len=%d want %d", len(schedule), len(countries))
	}
	if schedule[0].ISO2 != "PG" {
		t.Fatalf("first=%s want PG", schedule[0].ISO2)
	}
	seen := map[string]bool{}
	for _, c := range schedule {
		seen[c.ISO2] = true
	}
	for _, c := range countries {
		if !seen[c.ISO2] {
			t.Fatalf("missing %s in schedule", c.ISO2)
		}
	}
}

func TestCountrySchedule_unknownCountry(t *testing.T) {
	_, err := countrySchedule([]config.PacificCountry{{ISO2: "FJ"}}, "ZZ")
	if err == nil {
		t.Fatal("expected error for unknown country")
	}
}
