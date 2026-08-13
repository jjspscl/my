package timeutil

import (
	"fmt"
	"time"
)

// Clock resolves calendar dates in a fixed timezone. Every "today", month
// boundary, and date-range computation must go through a Clock so aggregation
// matches the user's financial calendar rather than the server's UTC clock.
type Clock struct {
	loc *time.Location
}

// New returns a Clock pinned to loc. A nil loc falls back to UTC.
func New(loc *time.Location) *Clock {
	if loc == nil {
		loc = time.UTC
	}
	return &Clock{loc: loc}
}

// Now returns the current instant expressed in the clock's location.
func (c *Clock) Now() time.Time {
	return time.Now().In(c.loc)
}

// TodayStart returns midnight at the start of today in the clock's location.
func (c *Clock) TodayStart() time.Time {
	now := c.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, c.loc)
}

// TodayStr returns today's date as YYYY-MM-DD in the clock's location.
func (c *Clock) TodayStr() string {
	return c.TodayStart().Format("2006-01-02")
}

// CurrentMonth returns the current month as YYYY-MM in the clock's location.
func (c *Clock) CurrentMonth() string {
	return c.Now().Format("2006-01")
}

// ParseDate parses a YYYY-MM-DD string into a time.Time in the clock's location.
func (c *Clock) ParseDate(s string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02", s, c.loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q, expected YYYY-MM-DD: %w", s, err)
	}
	return t, nil
}

// MonthRange returns the half-open [start, end) range for the given YYYY-MM
// month in the clock's location. The end is midnight on the first day of the
// following month, which makes it safe to use with `transaction_date >= ? AND
// transaction_date < ?` predicates.
func (c *Clock) MonthRange(month string) (time.Time, time.Time, error) {
	t, err := time.ParseInLocation("2006-01", month, c.loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid month %q, expected YYYY-MM: %w", month, err)
	}
	start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, c.loc)
	return start, start.AddDate(0, 1, 0), nil
}
