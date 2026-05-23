package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTransaction_Valid(t *testing.T) {
	tx, err := NewTransaction("tx-1", "user@test.com", "PHP", "food", "Lunch", 150000, TransactionExpense, time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(150000), tx.AmountCents)
	assert.Equal(t, TransactionExpense, tx.Type)
	assert.Equal(t, "food", tx.Category)
	assert.Equal(t, "PHP", tx.Currency)
}

func TestNewTransaction_Income(t *testing.T) {
	tx, err := NewTransaction("tx-2", "user@test.com", "PHP", "salary", "", 5000000, TransactionIncome, time.Now())
	require.NoError(t, err)
	assert.Equal(t, TransactionIncome, tx.Type)
}

func TestNewTransaction_ZeroAmount_Error(t *testing.T) {
	_, err := NewTransaction("tx-3", "user@test.com", "PHP", "food", "", 0, TransactionExpense, time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "positive")
}

func TestNewTransaction_NegativeAmount_Error(t *testing.T) {
	_, err := NewTransaction("tx-4", "user@test.com", "PHP", "food", "", -100, TransactionExpense, time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "positive")
}

func TestNewTransaction_EmptyCategory_Error(t *testing.T) {
	_, err := NewTransaction("tx-5", "user@test.com", "PHP", "", "", 100, TransactionExpense, time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "category")
}

func TestNewTransaction_CategoryTooLong_Error(t *testing.T) {
	longCat := make([]byte, MaxCategoryLen+1)
	for i := range longCat {
		longCat[i] = 'a'
	}
	_, err := NewTransaction("tx-6", "user@test.com", "PHP", string(longCat), "", 100, TransactionExpense, time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "category too long")
}

func TestNewTransaction_DescriptionTooLong_Error(t *testing.T) {
	longDesc := make([]byte, MaxDescriptionLen+1)
	for i := range longDesc {
		longDesc[i] = 'x'
	}
	_, err := NewTransaction("tx-7", "user@test.com", "PHP", "food", string(longDesc), 100, TransactionExpense, time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "description too long")
}

func TestNewTransaction_InvalidType_Error(t *testing.T) {
	_, err := NewTransaction("tx-8", "user@test.com", "PHP", "food", "", 100, "invalid", time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transaction type")
}

func TestTransactionType_Constants(t *testing.T) {
	assert.Equal(t, TransactionType("expense"), TransactionExpense)
	assert.Equal(t, TransactionType("income"), TransactionIncome)
	assert.NotEqual(t, TransactionExpense, TransactionIncome)
}

func TestDailyTotal_DefaultValues(t *testing.T) {
	total := DailyTotal{Date: "2026-01-15", Currency: "PHP"}
	assert.Equal(t, int64(0), total.TotalCents)
	assert.Equal(t, int64(0), total.ExpenseCents)
}
