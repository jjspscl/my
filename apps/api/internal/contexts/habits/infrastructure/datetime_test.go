package infrastructure

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseDatetimeHabit_DateOnly(t *testing.T) {
	result, err := parseDatetimeHabit("2026-01-15")
	assert.NoError(t, err)
	assert.Equal(t, 2026, result.Year())
	assert.Equal(t, time.January, result.Month())
	assert.Equal(t, 15, result.Day())
}

func TestParseDatetimeHabit_DateTime(t *testing.T) {
	result, err := parseDatetimeHabit("2026-01-15 14:30:00")
	assert.NoError(t, err)
	assert.Equal(t, 14, result.Hour())
	assert.Equal(t, 30, result.Minute())
}

func TestParseDatetimeHabit_ISO8601Z(t *testing.T) {
	result, err := parseDatetimeHabit("2026-01-15T14:30:00Z")
	assert.NoError(t, err)
	assert.Equal(t, 2026, result.Year())
}

func TestParseDatetimeHabit_RFC3339(t *testing.T) {
	result, err := parseDatetimeHabit("2026-01-15T14:30:00+08:00")
	assert.NoError(t, err)
	assert.Equal(t, 6, result.UTC().Hour()) // 14:30 +08:00 = 06:30 UTC
}

func TestParseDatetimeHabit_InvalidString_ReturnsError(t *testing.T) {
	_, err := parseDatetimeHabit("not-a-date")
	assert.Error(t, err)
}

func TestParseDatetimeHabit_EmptyString_ReturnsError(t *testing.T) {
	_, err := parseDatetimeHabit("")
	assert.Error(t, err)
}

func TestParseDatetimeHabit_LeapYear(t *testing.T) {
	result, err := parseDatetimeHabit("2024-02-29")
	assert.NoError(t, err)
	assert.Equal(t, 29, result.Day())
}

func TestParseDatetimeHabit_MonthDayBoundary(t *testing.T) {
	result, err := parseDatetimeHabit("2026-12-31 23:59:59")
	assert.NoError(t, err)
	assert.Equal(t, 31, result.Day())
	assert.Equal(t, time.December, result.Month())
	assert.Equal(t, 23, result.Hour())
	assert.Equal(t, 59, result.Minute())
	assert.Equal(t, 59, result.Second())
}

func TestParseDatetimeHabit_Midnight(t *testing.T) {
	result, err := parseDatetimeHabit("2026-01-01 00:00:00")
	assert.NoError(t, err)
	assert.Equal(t, 0, result.Hour())
	assert.Equal(t, 0, result.Minute())
	assert.Equal(t, 0, result.Second())
}

func TestParseDatetimeHabit_LeadingZeros(t *testing.T) {
	result, err := parseDatetimeHabit("2026-01-01 09:05:03")
	assert.NoError(t, err)
	assert.Equal(t, 9, result.Hour())
	assert.Equal(t, 5, result.Minute())
	assert.Equal(t, 3, result.Second())
}
