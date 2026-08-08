package domain

import (
	"fmt"
	"time"
)

type TransactionType string

const (
	TransactionExpense TransactionType = "expense"
	TransactionIncome  TransactionType = "income"

	MaxCategoryLen    = 100
	MaxDescriptionLen = 500
	MaxIdempotencyLen = 100
)

type Transaction struct {
	ID              string
	UserEmail       string
	AmountCents     int64
	Currency        string
	Category        string
	Description     string
	Type            TransactionType
	WalletID        string
	WalletName      string
	TransactionDate time.Time
	CreatedAt       time.Time
	IdempotencyKey  string
}

type DailyTotal struct {
	Date         string
	TotalCents   int64
	ExpenseCents int64
	IncomeCents  int64
	Currency     string
}

// CurrencyTotal is a per-currency aggregation. Analytics must never sum across
// currencies without an explicit conversion rate, so aggregates return one
// CurrencyTotal per currency present in the range.
type CurrencyTotal struct {
	Currency     string
	TotalCents   int64
	ExpenseCents int64
	IncomeCents  int64
}

// NewTransaction creates a validated transaction. Returns error if invariants fail.
func NewTransaction(id, userEmail, currency, category, description string, amountCents int64, txType TransactionType, txDate time.Time) (*Transaction, error) {
	if amountCents <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}
	if category == "" {
		return nil, fmt.Errorf("category is required")
	}
	if len(category) > MaxCategoryLen {
		return nil, fmt.Errorf("category too long (max %d)", MaxCategoryLen)
	}
	if len(description) > MaxDescriptionLen {
		return nil, fmt.Errorf("description too long (max %d)", MaxDescriptionLen)
	}
	if txType != TransactionExpense && txType != TransactionIncome {
		return nil, fmt.Errorf("invalid transaction type: %s", txType)
	}

	return &Transaction{
		ID:              id,
		UserEmail:       userEmail,
		AmountCents:     amountCents,
		Currency:        currency,
		Category:        category,
		Description:     description,
		Type:            txType,
		TransactionDate: txDate,
		CreatedAt:       time.Now().UTC(),
	}, nil
}
