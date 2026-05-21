package rampscore

import "testing"

func TestDMARCScorePct(t *testing.T) {
	tests := []struct {
		state, p, sp string
		wantPct      float64
		wantData     bool
	}{
		{"absent", "", "", 0, true},
		{"present", "none", "", 25, true},
		{"present", "quarantine", "", 75, true},
		{"present", "reject", "", 100, true},
		{"present", "none", "reject", 100, true},
		{"error", "", "", 0, false},
	}
	for _, tc := range tests {
		pct, ok := DMARCScorePct(tc.state, tc.p, tc.sp)
		if ok != tc.wantData || pct != tc.wantPct {
			t.Errorf("DMARCScorePct(%q,%q,%q) = %v,%v want %v,%v", tc.state, tc.p, tc.sp, pct, ok, tc.wantPct, tc.wantData)
		}
	}
}

func TestRPKIScorePct(t *testing.T) {
	pct, ok := RPKIScorePct(8, 0, 2, 10, "")
	if !ok || pct != 80 {
		t.Fatalf("partial valid: got %v %v", pct, ok)
	}
	pct, ok = RPKIScorePct(3, 1, 0, 4, "")
	if !ok || pct != 0 {
		t.Fatalf("any invalid: got %v %v", pct, ok)
	}
	_, ok = RPKIScorePct(0, 0, 0, 0, "")
	if ok {
		t.Fatal("expected no data for total=0")
	}
}

func TestRPKIWorstStatus(t *testing.T) {
	if RPKIWorstStatus(1, 1, 0) != "invalid" {
		t.Fatal("invalid wins")
	}
	if RPKIWorstStatus(0, 0, 3) != "unknown" {
		t.Fatal("all unknown")
	}
}
