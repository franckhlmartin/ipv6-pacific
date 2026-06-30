package ipv4outage

import "time"

const preOutageNoticeDays = 7

// InOutageWindow reports whether t is on UTC calendar day 6 (monthly drill).
func InOutageWindow(t time.Time) bool {
	return t.UTC().Day() == 6
}

// nextOutageStart is 00:00 UTC on the next calendar day 6 (this month or the next).
func nextOutageStart(t time.Time) time.Time {
	u := t.UTC()
	y, m, _ := u.Date()
	start := time.Date(y, m, 6, 0, 0, 0, 0, time.UTC)
	if !u.Before(start) {
		start = time.Date(y, m+1, 6, 0, 0, 0, 0, time.UTC)
	}
	return start
}

func calendarDaysUntil(from, until time.Time) int {
	fu := from.UTC()
	uu := until.UTC()
	fy, fm, fd := fu.Date()
	uy, um, ud := uu.Date()
	fromDay := time.Date(fy, fm, fd, 0, 0, 0, 0, time.UTC)
	untilDay := time.Date(uy, um, ud, 0, 0, 0, 0, time.UTC)
	return int(untilDay.Sub(fromDay).Hours() / 24)
}

// InPreOutageWindow reports whether t is within the 7 calendar days before the next UTC day 6.
func InPreOutageWindow(t time.Time) bool {
	days := calendarDaysUntil(t, nextOutageStart(t))
	return days >= 1 && days <= preOutageNoticeDays
}

// UnavailableUntil is the instant IPv4 service may resume (start of next UTC day).
func UnavailableUntil(t time.Time) time.Time {
	u := t.UTC()
	y, m, d := u.Date()
	return time.Date(y, m, d+1, 0, 0, 0, 0, time.UTC)
}

// DaysUntilOutage returns calendar days until the next UTC day 6 from t (for pre-outage banner).
func DaysUntilOutage(t time.Time) int {
	if !InPreOutageWindow(t) {
		return 0
	}
	return calendarDaysUntil(t, nextOutageStart(t))
}

// OutageActive returns whether policy should treat the outage as enabled at t.
func OutageActive(cfg Config, t time.Time) bool {
	if cfg.Skip {
		return false
	}
	return InOutageWindow(t) || cfg.Force
}
