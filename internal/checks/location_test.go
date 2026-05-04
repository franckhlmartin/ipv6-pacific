package checks

import "testing"

func TestClassifyLocation(t *testing.T) {
	if g := classifyLocation("ns1.gov.fj", "gov.fj"); g != "I" {
		t.Fatalf("got %s", g)
	}
	if g := classifyLocation("ns.example.com", "gov.fj"); g != "O" {
		t.Fatalf("got %s", g)
	}
}
