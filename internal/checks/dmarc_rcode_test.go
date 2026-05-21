package checks

import (
	"errors"
	"testing"
)

func TestAbsentDMARC(t *testing.T) {
	col := absentDMARC()
	if col.State != "absent" || col.Display != "No record" {
		t.Fatalf("got %+v", col)
	}
	if col.ScorePct != 0 {
		t.Fatalf("score %v", col.ScorePct)
	}
}

func TestDNSNotFound(t *testing.T) {
	if !dnsNotFound(errors.New("read udp: NXDOMAIN")) {
		t.Fatal("expected NXDOMAIN as not found")
	}
	if dnsNotFound(errors.New("timeout")) {
		t.Fatal("timeout is not not-found")
	}
}
