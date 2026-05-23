package infrastructure

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseDatetime_DateOnly(t *testing.T) {
	result, err := parseDatetime("2026-01-15")
	assert.NoError(t, err)
	assert.Equal(t, 2026, result.Year())
	assert.Equal(t, time.January, result.Month())
	assert.Equal(t, 15, result.Day())
	assert.Equal(t, 0, result.Hour())
	assert.Equal(t, 0, result.Minute())
	assert.Equal(t, 0, result.Second())
}

func TestParseDatetime_DateTime(t *testing.T) {
	result, err := parseDatetime("2026-01-15 14:30:00")
	assert.NoError(t, err)
	assert.Equal(t, 2026, result.Year())
	assert.Equal(t, time.January, result.Month())
	assert.Equal(t, 15, result.Day())
	assert.Equal(t, 14, result.Hour())
	assert.Equal(t, 30, result.Minute())
	assert.Equal(t, 0, result.Second())
}

func TestParseDatetime_ISO8601Z(t *testing.T) {
	result, err := parseDatetime("2026-01-15T14:30:00Z")
	assert.NoError(t, err)
	assert.Equal(t, 2026, result.Year())
	assert.Equal(t, 14, result.Hour())
}

func TestParseDatetime_RFC3339(t *testing.T) {
	result, err := parseDatetime("2026-01-15T14:30:00+08:00")
	assert.NoError(t, err)
	assert.Equal(t, 2026, result.Year())
	// +08:00 offset means UTC hour is 6
	assert.Equal(t, 6, result.UTC().Hour())
}

func TestParseDatetime_InvalidString_ReturnsError(t *testing.T) {
	_, err := parseDatetime("not-a-date")
	assert.Error(t, err)
}

func TestParseDatetime_EmptyString_ReturnsError(t *testing.T) {
	_, err := parseDatetime("")
	assert.Error(t, err)
}

func TestParseDatetime_LeapYear(t *testing.T) {
	result, err := parseDatetime("2024-02-29")
	assert.NoError(t, err)
	assert.Equal(t, 29, result.Day())
	assert.Equal(t, time.February, result.Month())
}

func TestParseDatetime_DifferentDelimiters(t *testing.T) {
	// Only standard formats should parse
	_, err := parseDatetime("2026/01/15")
	assert.Error(t, err)
}

func TestParseDatetime_SecondPrecision(t *testing.T) {
	result, err := parseDatetime("2026-06-15 09:05:30")
	assert.NoError(t, err)
	assert.Equal(t, 30, result.Second())
	assert.Equal(t, 9, result.Hour())
	assert.Equal(t, 5, result.Minute())
}

func TestParseDatetime_Midnight(t *testing.T) {
	result, err := parseDatetime("2026-01-01 00:00:00")
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Day())
	assert.Equal(t, time.January, result.Month())
	assert.Equal(t, 0, result.Hour())
	assert.Equal(t, 0, result.Minute())
	assert.Equal(t, 0, result.Second())
}
