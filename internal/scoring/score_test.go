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
	if got := RowScore(d); got != 0+1+1+1 {
		t.Fatalf("RowScore = %d want 3", got)
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

func TestEconomyDeploymentScorePct(t *testing.T) {
	maxed := model.DomainResult{
		DNS:    model.ServiceColumn{Class: model.DeployIPv6Only},
		Mail:   model.ServiceColumn{Class: model.DeployIPv6Only},
		Web:    model.ServiceColumn{Class: model.DeployIPv6Only},
		DNSSEC: model.DNSSECColumn{State: "signed"},
	}
	if EconomyDeploymentScorePct(nil) != 0 {
		t.Fatalf("empty want 0")
	}
	if got := EconomyDeploymentScorePct([]model.DomainResult{maxed}); got != 100 {
		t.Fatalf("one maxed domain = %v want 100", got)
	}
	if got := EconomyDeploymentScorePct([]model.DomainResult{maxed, maxed}); got != 100 {
		t.Fatalf("two maxed = %v want 100", got)
	}
	half := model.DomainResult{
		DNS:    model.ServiceColumn{Class: model.DeployDual},
		Mail:   model.ServiceColumn{Class: model.DeployDual},
		Web:    model.ServiceColumn{Class: model.DeployDual},
		DNSSEC: model.DNSSECColumn{State: "unsigned"},
	}
	// RowScore = 1+1+1+0 = 3; 3/4 * 100 = 75
	if got := EconomyDeploymentScorePct([]model.DomainResult{half}); got != 75 {
		t.Fatalf("half-ish row = %v want 75", got)
	}
}
