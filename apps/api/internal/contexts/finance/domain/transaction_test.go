package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTransaction_ValidCreation(t *testing.T) {
	tx := &Transaction{
		ID:              "tx-1",
		UserEmail:       "user@test.com",
		AmountCents:     150000,
		Currency:        "PHP",
		Category:        "food",
		Description:     "Lunch",
		Type:            TransactionExpense,
		TransactionDate: time.Now(),
	}

	assert.Equal(t, int64(150000), tx.AmountCents)
	assert.Equal(t, TransactionExpense, tx.Type)
	assert.Equal(t, "food", tx.Category)
}

func TestTransaction_IncomeType(t *testing.T) {
	tx := &Transaction{
		ID:              "tx-2",
		UserEmail:       "user@test.com",
		AmountCents:     5000000,
		Currency:        "PHP",
		Category:        "salary",
		Type:            TransactionIncome,
		TransactionDate: time.Now(),
	}

	assert.Equal(t, TransactionIncome, tx.Type)
	assert.Equal(t, int64(5000000), tx.AmountCents)
}

func TestTransaction_ZeroAmount(t *testing.T) {
	tx := &Transaction{
		AmountCents: 0,
		Type:        TransactionExpense,
	}

	// Domain object itself doesn't validate, but the service does
	// Test the domain object carries the value correctly
	assert.Equal(t, int64(0), tx.AmountCents)
}

func TestTransaction_NegativeAmount(t *testing.T) {
	tx := &Transaction{
		AmountCents: -100,
		Type:        TransactionExpense,
	}

	assert.Equal(t, int64(-100), tx.AmountCents)
}

func TestTransactionType_Constants(t *testing.T) {
	assert.Equal(t, TransactionType("expense"), TransactionExpense)
	assert.Equal(t, TransactionType("income"), TransactionIncome)
	assert.NotEqual(t, TransactionExpense, TransactionIncome)
}

func TestDailyTotal_DefaultValues(t *testing.T) {
	total := DailyTotal{
		Date:     "2026-01-15",
		Currency: "PHP",
	}

	assert.Equal(t, "2026-01-15", total.Date)
	assert.Equal(t, "PHP", total.Currency)
	assert.Equal(t, int64(0), total.TotalCents)
	assert.Equal(t, int64(0), total.ExpenseCents)
	assert.Equal(t, int64(0), total.IncomeCents)
}

func TestDailyTotal_WithValues(t *testing.T) {
	total := DailyTotal{
		Date:         "2026-01-15",
		TotalCents:   50000,
		ExpenseCents: 20000,
		IncomeCents:  70000,
		Currency:     "PHP",
	}

	assert.Equal(t, int64(50000), total.TotalCents)
	assert.Equal(t, int64(20000), total.ExpenseCents)
	assert.Equal(t, int64(70000), total.IncomeCents)
}
