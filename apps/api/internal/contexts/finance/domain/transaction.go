package domain

import "time"

type TransactionType string

const (
	TransactionExpense TransactionType = "expense"
	TransactionIncome  TransactionType = "income"
)

type Transaction struct {
	ID              string
	UserEmail       string
	AmountCents     int64
	Currency        string
	Category        string
	Description     string
	Type            TransactionType
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
