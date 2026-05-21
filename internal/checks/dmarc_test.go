package checks

import "testing"

func TestParseDMARCTags(t *testing.T) {
	tags := parseDMARCTags("v=DMARC1; p=reject; sp=quarantine; rua=mailto:a@b.c")
	if tags["p"] != "reject" || tags["sp"] != "quarantine" {
		t.Fatalf("tags: %v", tags)
	}
}

func TestEffectiveSP(t *testing.T) {
	if effectiveSP("none", "") != "none" {
		t.Fatal("inherit p")
	}
	if effectiveSP("none", "reject") != "reject" {
		t.Fatal("explicit sp")
	}
}

func TestDMARCDisplay(t *testing.T) {
	if dmarcDisplay("none", "reject", "reject") != "p=none, sp=reject" {
		t.Fatalf("display with distinct sp")
	}
	if dmarcDisplay("reject", "", "reject") != "reject" {
		t.Fatalf("display p only")
	}
}
