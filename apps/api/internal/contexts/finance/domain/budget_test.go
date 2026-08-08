package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBudget_Valid(t *testing.T) {
	b, err := NewBudget("b-1", "user@test.com", "2026-05", "PHP")
	require.NoError(t, err)
	assert.Equal(t, "2026-05", b.Month)
	assert.Equal(t, "user@test.com", b.UserEmail)
	assert.NotZero(t, b.CreatedAt)
}

func TestNewBudget_InvalidMonth(t *testing.T) {
	_, err := NewBudget("b-2", "user@test.com", "2026-13", "PHP")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid month format")
}

func TestNewBudget_InvalidMonthFormat(t *testing.T) {
	_, err := NewBudget("b-3", "user@test.com", "May 2026", "PHP")
	assert.Error(t, err)
}

func TestNewBudget_EmptyEmail(t *testing.T) {
	_, err := NewBudget("b-4", "", "2026-05", "PHP")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user email")
}

func TestNewBudgetCategory_Valid(t *testing.T) {
	bc, err := NewBudgetCategory("bc-1", "b-1", "Food", 500000, false)
	require.NoError(t, err)
	assert.Equal(t, int64(500000), bc.AllocatedCents)
	assert.False(t, bc.RolloverEnabled)
}

func TestNewBudgetCategory_WithRollover(t *testing.T) {
	bc, err := NewBudgetCategory("bc-2", "b-1", "Savings", 1000000, true)
	require.NoError(t, err)
	assert.True(t, bc.RolloverEnabled)
}

func TestNewBudgetCategory_NegativeAllocation(t *testing.T) {
	_, err := NewBudgetCategory("bc-3", "b-1", "Food", -100, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "negative")
}

func TestNewBudgetCategory_EmptyCategory(t *testing.T) {
	_, err := NewBudgetCategory("bc-4", "b-1", "", 100, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "category is required")
}

func TestNewBudgetCategory_TooLong(t *testing.T) {
	long := make([]byte, MaxCategoryLen+1)
	for i := range long {
		long[i] = 'a'
	}
	_, err := NewBudgetCategory("bc-5", "b-1", string(long), 100, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too long")
}

func TestNewBudgetCategory_ZeroAllocation(t *testing.T) {
	bc, err := NewBudgetCategory("bc-6", "b-1", "Misc", 0, false)
	require.NoError(t, err)
	assert.Equal(t, int64(0), bc.AllocatedCents)
}
