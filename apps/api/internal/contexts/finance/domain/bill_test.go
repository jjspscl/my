package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRecurringBill_Valid(t *testing.T) {
	b, err := NewRecurringBill("b-1", "user@test.com", "Netflix", "Subscription", 49900, "PHP", FrequencyMonthly, 15, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), nil, false, nil)
	require.NoError(t, err)
	assert.Equal(t, "Netflix", b.Name)
	assert.Equal(t, int64(49900), b.AmountCents)
	assert.Equal(t, FrequencyMonthly, b.Frequency)
	assert.Equal(t, 15, b.DayOfMonth)
	assert.NotZero(t, b.CreatedAt)
}

func TestNewRecurringBill_WithEndDate(t *testing.T) {
	end := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	b, err := NewRecurringBill("b-2", "user@test.com", "Rent", "Housing", 1500000, "PHP", FrequencyMonthly, 1, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &end, true, strPtr("RENT"))
	require.NoError(t, err)
	assert.True(t, b.AutoMatch)
	assert.Equal(t, "RENT", *b.MatchPattern)
	assert.NotNil(t, b.EndDate)
}

func TestNewRecurringBill_Weekly(t *testing.T) {
	b, err := NewRecurringBill("b-3", "user@test.com", "Grocery", "Food", 500000, "PHP", FrequencyWeekly, 0, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), nil, false, nil)
	require.NoError(t, err)
	assert.Equal(t, FrequencyWeekly, b.Frequency)
	assert.Equal(t, 0, b.DayOfMonth)
}

func TestNewRecurringBill_EmptyName(t *testing.T) {
	_, err := NewRecurringBill("b-4", "user@test.com", "  ", "Cat", 100, "PHP", FrequencyMonthly, 1, time.Now(), nil, false, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestNewRecurringBill_EmptyCategory(t *testing.T) {
	_, err := NewRecurringBill("b-5", "user@test.com", "Test", "", 100, "PHP", FrequencyMonthly, 1, time.Now(), nil, false, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "category is required")
}

func TestNewRecurringBill_ZeroAmount(t *testing.T) {
	_, err := NewRecurringBill("b-6", "user@test.com", "Test", "Cat", 0, "PHP", FrequencyMonthly, 1, time.Now(), nil, false, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be positive")
}

func TestNewRecurringBill_InvalidFrequency(t *testing.T) {
	_, err := NewRecurringBill("b-7", "user@test.com", "Test", "Cat", 100, "PHP", Frequency("biweekly"), 1, time.Now(), nil, false, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid frequency")
}

func TestNewRecurringBill_DayOfMonthOutOfRange(t *testing.T) {
	_, err := NewRecurringBill("b-8", "user@test.com", "Test", "Cat", 100, "PHP", FrequencyMonthly, 32, time.Now(), nil, false, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestNewRecurringBill_EndDateBeforeStart(t *testing.T) {
	end := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	_, err := NewRecurringBill("b-9", "user@test.com", "Test", "Cat", 100, "PHP", FrequencyMonthly, 1, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &end, false, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "end date must be after start date")
}

func TestNewRecurringBill_NameTooLong(t *testing.T) {
	long := make([]byte, MaxBillNameLen+1)
	for i := range long {
		long[i] = 'a'
	}
	_, err := NewRecurringBill("b-10", "user@test.com", string(long), "Cat", 100, "PHP", FrequencyMonthly, 1, time.Now(), nil, false, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name too long")
}

func TestNewBillPayment_Valid(t *testing.T) {
	p, err := NewBillPayment("p-1", "b-1", time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC), 49900)
	require.NoError(t, err)
	assert.Equal(t, OccurrencePending, p.Status)
	assert.Equal(t, "b-1", p.BillID)
	assert.NotZero(t, p.CreatedAt)
}

func TestNewBillPayment_EmptyBillID(t *testing.T) {
	_, err := NewBillPayment("p-2", "", time.Now(), 100)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bill id is required")
}

func TestNewBillPayment_ZeroAmount(t *testing.T) {
	_, err := NewBillPayment("p-3", "b-1", time.Now(), 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be positive")
}

func strPtr(s string) *string {
	return &s
}
