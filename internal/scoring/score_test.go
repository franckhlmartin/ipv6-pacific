package scoring

import (
	"testing"

	"github.com/pacific-monitor/pacific-monitor/internal/model"
)

func TestRowScore(t *testing.T) {
	d := model.DomainResult{
		DNS:    model.ServiceColumn{Class: model.DeployIPv4Only},
		Mail:   model.ServiceColumn{Class: model.DeployDual},
		Web:    model.ServiceColumn{Class: model.DeployIPv6Only},
		DNSSEC: model.DNSSECColumn{State: "signed"},
	}
	if got := RowScore(d); got != 0+1+2+2 {
		t.Fatalf("RowScore = %d want 5", got)
	}
}

func TestDNSSECCellClass(t *testing.T) {
	if got := DNSSECCellClass(model.DNSSECColumn{State: "signed"}); got != "ipv6_only" {
		t.Fatalf("signed: %q", got)
	}
	if got := DNSSECCellClass(model.DNSSECColumn{State: "unsigned"}); got != "ipv4_only" {
		t.Fatalf("unsigned: %q", got)
	}
	if got := DNSSECCellClass(model.DNSSECColumn{State: "error"}); got != "unknown" {
		t.Fatalf("error: %q", got)
	}
}

func TestRowScoreDNSSECUnsigned(t *testing.T) {
	d := model.DomainResult{
		DNS:    model.ServiceColumn{Class: model.DeployUnknown},
		Mail:   model.ServiceColumn{Class: model.DeployUnknown},
		Web:    model.ServiceColumn{Class: model.DeployUnknown},
		DNSSEC: model.DNSSECColumn{State: "unsigned"},
	}
	if got := RowScore(d); got != 0 {
		t.Fatalf("RowScore = %d want 0", got)
	}
}
