package ipv4outage

import "time"

// InOutageWindow reports whether t is on UTC calendar day 6 (monthly drill).
func InOutageWindow(t time.Time) bool {
	return t.UTC().Day() == 6
}

// InPreOutageWindow reports UTC days 4–5 (advance notice window).
func InPreOutageWindow(t time.Time) bool {
	d := t.UTC().Day()
	return d == 4 || d == 5
}

// UnavailableUntil is the instant IPv4 service may resume (start of next UTC day).
func UnavailableUntil(t time.Time) time.Time {
	u := t.UTC()
	y, m, d := u.Date()
	return time.Date(y, m, d+1, 0, 0, 0, 0, time.UTC)
}

// DaysUntilOutage returns calendar days until the next UTC day 6 from t (for pre-outage banner).
// On day 4 returns 2; on day 5 returns 1; otherwise 0.
func DaysUntilOutage(t time.Time) int {
	if !InPreOutageWindow(t) {
		return 0
	}
	return 6 - t.UTC().Day()
}

// OutageActive returns whether policy should treat the outage as enabled at t.
func OutageActive(cfg Config, t time.Time) bool {
	if cfg.Skip {
		return false
	}
	return InOutageWindow(t) || cfg.Force
}
