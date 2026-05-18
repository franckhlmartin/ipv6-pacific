package apnicstats

// ASNRow is one ASN from the APNIC Labs country stats table (stats.labs.apnic.net/ipv6/{CC}).
type ASNRow struct {
	ASN              string
	ASNNumber        int
	Name             string
	IPv6CapablePct   float64
	IPv6PreferredPct float64
	Samples          int
}
