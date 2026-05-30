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
}

type DailyTotal struct {
	Date         string
	TotalCents   int64
	ExpenseCents int64
	IncomeCents  int64
	Currency     string
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
