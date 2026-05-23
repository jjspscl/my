package domain

import (
	"context"
	"time"
)

type TransactionRepository interface {
	Save(ctx context.Context, tx *Transaction) error
	FindByID(ctx context.Context, id string) (*Transaction, error)
	ListByUserAndDateRange(ctx context.Context, userEmail string, from, to time.Time, limit, offset int) ([]*Transaction, error)
	Delete(ctx context.Context, id, userEmail string) error
	GetTodayTotal(ctx context.Context, userEmail string, date time.Time) (*DailyTotal, error)
}
