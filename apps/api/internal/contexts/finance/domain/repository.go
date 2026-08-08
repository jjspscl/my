package domain

import (
	"context"
	"time"
)

type WalletRepository interface {
	Save(ctx context.Context, wallet *Wallet) error
	FindByID(ctx context.Context, id string) (*Wallet, error)
	ListByUser(ctx context.Context, userEmail string) ([]*Wallet, error)
	Update(ctx context.Context, wallet *Wallet) error
	Archive(ctx context.Context, id, userEmail string) error
	FindDefault(ctx context.Context, userEmail string) (*Wallet, error)
	GetBalancesByUser(ctx context.Context, userEmail string) ([]*WalletBalance, error)
}

type TransferRepository interface {
	Save(ctx context.Context, transfer *WalletTransfer) error
	FindByID(ctx context.Context, id string) (*WalletTransfer, error)
	FindByIdempotencyKey(ctx context.Context, userEmail, key string) (*WalletTransfer, error)
	ListByUser(ctx context.Context, userEmail string, limit, offset int) ([]*WalletTransfer, error)
}

type TransactionRepository interface {
	Save(ctx context.Context, tx *Transaction) error
	FindByID(ctx context.Context, id string) (*Transaction, error)
	FindByIdempotencyKey(ctx context.Context, userEmail, key string) (*Transaction, error)
	ListByUserAndDateRange(ctx context.Context, userEmail string, from, to time.Time, limit, offset int) ([]*Transaction, error)
	Delete(ctx context.Context, id, userEmail string) error
	GetTodayTotals(ctx context.Context, userEmail string, date time.Time) ([]CurrencyTotal, error)
}

type BudgetRepository interface {
	UpsertBudget(ctx context.Context, budget *Budget) error
	FindBudgetByMonth(ctx context.Context, userEmail, month string) (*Budget, error)
	UpsertBudgetCategories(ctx context.Context, budgetID string, categories []*BudgetCategory) error
	GetBudgetCategories(ctx context.Context, budgetID string) ([]*BudgetCategory, error)
	GetSpentByCategory(ctx context.Context, userEmail, currency string, from, to time.Time) (map[string]int64, error)
}

type BillRepository interface {
	SaveBill(ctx context.Context, bill *RecurringBill) error
	UpdateBill(ctx context.Context, bill *RecurringBill) error
	DeleteBill(ctx context.Context, id, userEmail string) error
	FindBillByID(ctx context.Context, id string) (*RecurringBill, error)
	ListBills(ctx context.Context, userEmail string) ([]*RecurringBill, error)
	SavePayment(ctx context.Context, payment *BillPayment) error
	FindPayment(ctx context.Context, billID, dueDate string) (*BillPayment, error)
	ListPaymentsByBill(ctx context.Context, billID string) ([]*BillPayment, error)
	ListPaymentsByBills(ctx context.Context, billIDs []string, from, to time.Time) ([]*BillPayment, error)
	ListUpcomingBills(ctx context.Context, userEmail string, limit int) ([]*BillWithPayment, error)
	FindTransactionByMatch(ctx context.Context, userEmail, category string, amountCents int64, date string, pattern string) (*Transaction, error)
}

type GoalRepository interface {
	SaveGoal(ctx context.Context, goal *SavingsGoal) error
	UpdateGoal(ctx context.Context, goal *SavingsGoal) error
	DeleteGoal(ctx context.Context, id, userEmail string) error
	FindGoalByID(ctx context.Context, id string) (*SavingsGoal, error)
	ListGoals(ctx context.Context, userEmail string) ([]*SavingsGoal, error)
	SaveContribution(ctx context.Context, contribution *GoalContribution) error
	FindContributionByIdempotencyKey(ctx context.Context, key string) (*GoalContribution, error)
	ListContributionsByGoal(ctx context.Context, goalID string) ([]*GoalContribution, error)
	GetCurrentAmountByGoal(ctx context.Context, goalID string) (int64, error)
	GetCurrentAmountsByGoals(ctx context.Context, goalIDs []string) (map[string]int64, error)
}
