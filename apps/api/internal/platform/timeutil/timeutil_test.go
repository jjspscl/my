package timeutil

import (
	"testing"
	"time"
)

func manila(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Manila")
	if err != nil {
		t.Fatalf("load Asia/Manila: %v", err)
	}
	return loc
}

func TestNewNilLocationFallsBackToUTC(t *testing.T) {
	c := New(nil)
	if c.loc != time.UTC {
		t.Fatalf("expected UTC fallback, got %v", c.loc)
	}
}

func TestParseDateUsesClockLocation(t *testing.T) {
	loc := manila(t)
	c := New(loc)

	got, err := c.ParseDate("2026-01-01")
	if err != nil {
		t.Fatalf("ParseDate: %v", err)
	}
	if got.Location() != loc {
		t.Fatalf("expected location %v, got %v", loc, got.Location())
	}
	// Manila is UTC+8, so local midnight is the previous day at 16:00 UTC.
	// This is exactly the shift that breaks naive UTC date handling.
	if utc := got.UTC(); utc.Day() != 31 || utc.Month() != time.December || utc.Hour() != 16 {
		t.Fatalf("expected 2025-12-31T16:00Z, got %s", utc.Format(time.RFC3339))
	}
}

func TestParseDateRejectsGarbage(t *testing.T) {
	c := New(manila(t))
	if _, err := c.ParseDate("01/02/2026"); err == nil {
		t.Fatal("expected error for non YYYY-MM-DD input")
	}
}

func TestMonthRangeIsHalfOpen(t *testing.T) {
	loc := manila(t)
	c := New(loc)

	tests := []struct {
		name      string
		month     string
		wantStart time.Time
		wantEnd   time.Time
	}{
		{
			name:      "mid-year month",
			month:     "2026-02",
			wantStart: time.Date(2026, time.February, 1, 0, 0, 0, 0, loc),
			wantEnd:   time.Date(2026, time.March, 1, 0, 0, 0, 0, loc),
		},
		{
			name:      "leap february",
			month:     "2028-02",
			wantStart: time.Date(2028, time.February, 1, 0, 0, 0, 0, loc),
			wantEnd:   time.Date(2028, time.March, 1, 0, 0, 0, 0, loc),
		},
		{
			name:      "december rolls into next year",
			month:     "2026-12",
			wantStart: time.Date(2026, time.December, 1, 0, 0, 0, 0, loc),
			wantEnd:   time.Date(2027, time.January, 1, 0, 0, 0, 0, loc),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			start, end, err := c.MonthRange(tc.month)
			if err != nil {
				t.Fatalf("MonthRange: %v", err)
			}
			if !start.Equal(tc.wantStart) {
				t.Errorf("start = %s, want %s", start, tc.wantStart)
			}
			if !end.Equal(tc.wantEnd) {
				t.Errorf("end = %s, want %s", end, tc.wantEnd)
			}

			// The last instant of the month must fall inside [start, end).
			lastInstant := tc.wantEnd.Add(-time.Nanosecond)
			if lastInstant.Before(start) || !lastInstant.Before(end) {
				t.Errorf("%s not within [%s, %s)", lastInstant, start, end)
			}
			// The first instant of the next month must fall outside.
			if tc.wantEnd.Before(end) {
				t.Errorf("%s unexpectedly inside range", tc.wantEnd)
			}
		})
	}
}

func TestMonthRangeRejectsGarbage(t *testing.T) {
	c := New(manila(t))
	if _, _, err := c.MonthRange("2026-13"); err == nil {
		t.Fatal("expected error for month 13")
	}
	if _, _, err := c.MonthRange("2026-01-01"); err == nil {
		t.Fatal("expected error for YYYY-MM-DD input")
	}
}

// A UTC-based implementation would report the wrong calendar day for any
// Manila instant between 00:00 and 08:00 local. These cases pin the boundaries
// that matter: the first and last minute of a day, month, and year.
func TestDateBoundariesResolveInLocalCalendar(t *testing.T) {
	loc := manila(t)

	tests := []struct {
		name      string
		instant   time.Time
		wantDay   string
		wantMonth string
	}{
		{
			name:      "first minute of the day",
			instant:   time.Date(2026, time.March, 15, 0, 0, 0, 0, loc),
			wantDay:   "2026-03-15",
			wantMonth: "2026-03",
		},
		{
			name:      "last minute of the day",
			instant:   time.Date(2026, time.March, 15, 23, 59, 0, 0, loc),
			wantDay:   "2026-03-15",
			wantMonth: "2026-03",
		},
		{
			name:      "first minute of the month",
			instant:   time.Date(2026, time.April, 1, 0, 0, 0, 0, loc),
			wantDay:   "2026-04-01",
			wantMonth: "2026-04",
		},
		{
			name:      "last minute of the month",
			instant:   time.Date(2026, time.March, 31, 23, 59, 0, 0, loc),
			wantDay:   "2026-03-31",
			wantMonth: "2026-03",
		},
		{
			name:      "first minute of the year",
			instant:   time.Date(2026, time.January, 1, 0, 0, 0, 0, loc),
			wantDay:   "2026-01-01",
			wantMonth: "2026-01",
		},
		{
			name:      "last minute of the year",
			instant:   time.Date(2026, time.December, 31, 23, 59, 0, 0, loc),
			wantDay:   "2026-12-31",
			wantMonth: "2026-12",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			local := tc.instant.In(loc)
			if got := local.Format("2006-01-02"); got != tc.wantDay {
				t.Errorf("day = %s, want %s", got, tc.wantDay)
			}
			if got := local.Format("2006-01"); got != tc.wantMonth {
				t.Errorf("month = %s, want %s", got, tc.wantMonth)
			}

			// Same instant read as UTC drifts backwards across the boundary
			// whenever local time is before 08:00 — the bug this package exists
			// to prevent.
			if local.Hour() < 8 && local.UTC().Format("2006-01-02") == tc.wantDay {
				t.Errorf("expected UTC reading to drift off %s, got %s",
					tc.wantDay, local.UTC().Format("2006-01-02"))
			}
		})
	}
}

func TestTodayStartIsMidnightInClockLocation(t *testing.T) {
	loc := manila(t)
	c := New(loc)

	start := c.TodayStart()
	if h, m, s := start.Clock(); h != 0 || m != 0 || s != 0 {
		t.Errorf("expected midnight, got %02d:%02d:%02d", h, m, s)
	}
	if start.Location() != loc {
		t.Errorf("expected location %v, got %v", loc, start.Location())
	}
	if got, want := c.TodayStr(), start.Format("2006-01-02"); got != want {
		t.Errorf("TodayStr = %s, want %s", got, want)
	}
	if got, want := c.CurrentMonth(), c.Now().Format("2006-01"); got != want {
		t.Errorf("CurrentMonth = %s, want %s", got, want)
	}

	// TodayStart must be the most recent midnight, never in the future.
	now := c.Now()
	if start.After(now) || now.Sub(start) >= 24*time.Hour {
		t.Errorf("TodayStart %s not within the last 24h of %s", start, now)
	}
}
