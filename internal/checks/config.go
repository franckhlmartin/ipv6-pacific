package checks

import "time"

// Config holds deadlines for a single domain check (subset from collector env).
type Config struct {
	DNSResolveTimeout time.Duration
	HTTPTimeout       time.Duration
	SMTPTimeout       time.Duration
	DomainDeadline    time.Duration

	// LogStep is called after each major phase completes (DNS, Mail, Web, DNSSEC).
	// phase is a short label, e.g. "DNS", "Mail", "Web", "DNSSEC".
	// timeoutDesc briefly lists which limits apply (e.g. DNSResolveTimeout=5s).
	// summary is a compact outcome string (class, display snippet, errors).
	LogStep func(phase string, timeoutDesc string, summary string)
}

// DefaultConfig uses conservative timeouts suitable for batch measurement.
func DefaultConfig() Config {
	return Config{
		DNSResolveTimeout: 5 * time.Second,
		HTTPTimeout:       15 * time.Second,
		SMTPTimeout:       10 * time.Second,
		DomainDeadline:    120 * time.Second,
	}
}
