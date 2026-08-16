package http

import (
	"context"
	"errors"
	"time"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
)

// Fake repositories for handler tests. Each fake holds settable return values
// plus an optional err that, when non-nil, is returned by every method — so a
// test can force a repository failure without touching the happy path. The
// fakes are deliberately trimmed to what the analytics handlers reach through
// their services; they are not general-purpose doubles.

const analyticsTestUser = "user@example.com"

// --- fakeAnalyticsRepo ---

type fakeAnalyticsRepo struct {
	err                error
	cashFlow           []domain.CurrencyTotal
	monthlyFlow        []domain.MonthlyCashFlow
	classification     []domain.ClassificationSpend
	unclassified       []domain.UnclassifiedSpending
	topUnclassified    []domain.CategorySpend
	categoryMonthly    []domain.MonthlyAmount
	unbudgeted         int64
	categoryMonthlyAll []domain.CategoryMonthlySpend
	expenseAmounts     []domain.ExpenseAmount
	billReconciliation []domain.BillReconciliationRow
	essentialMonthly   []domain.MonthlyEssentialSpend
}

func (f *fakeAnalyticsRepo) fail() error {
	if f.err != nil {
		return f.err
	}
	return nil
}

func (f *fakeAnalyticsRepo) GetCashFlow(_ context.Context, _ string, _, _ time.Time) ([]domain.CurrencyTotal, error) {
	return f.cashFlow, f.fail()
}
func (f *fakeAnalyticsRepo) GetMonthlyCashFlow(_ context.Context, _ string, _, _ time.Time) ([]domain.MonthlyCashFlow, error) {
	return f.monthlyFlow, f.fail()
}
func (f *fakeAnalyticsRepo) GetSpendingByClassification(_ context.Context, _ string, _, _ time.Time) ([]domain.ClassificationSpend, error) {
	return f.classification, f.fail()
}
func (f *fakeAnalyticsRepo) GetUnclassifiedSpending(_ context.Context, _ string, _, _ time.Time) ([]domain.UnclassifiedSpending, error) {
	return f.unclassified, f.fail()
}
func (f *fakeAnalyticsRepo) GetTopUnclassifiedCategories(_ context.Context, _ string, _, _ time.Time, _ int) ([]domain.CategorySpend, error) {
	return f.topUnclassified, f.fail()
}
func (f *fakeAnalyticsRepo) GetCategoryMonthlySpend(_ context.Context, _, _, _ string, _, _ time.Time) ([]domain.MonthlyAmount, error) {
	return f.categoryMonthly, f.fail()
}
func (f *fakeAnalyticsRepo) GetUnbudgetedSpend(_ context.Context, _, _, _ string, _, _ time.Time) (int64, error) {
	return f.unbudgeted, f.fail()
}
func (f *fakeAnalyticsRepo) GetCategoryMonthlySpendAll(_ context.Context, _ string, _, _ time.Time) ([]domain.CategoryMonthlySpend, error) {
	return f.categoryMonthlyAll, f.fail()
}
func (f *fakeAnalyticsRepo) GetExpenseAmounts(_ context.Context, _ string, _, _ time.Time) ([]domain.ExpenseAmount, error) {
	return f.expenseAmounts, f.fail()
}
func (f *fakeAnalyticsRepo) GetBillReconciliation(_ context.Context, _ string, _, _ time.Time) ([]domain.BillReconciliationRow, error) {
	return f.billReconciliation, f.fail()
}
func (f *fakeAnalyticsRepo) GetEssentialMonthlySpend(_ context.Context, _ string, _, _ time.Time) ([]domain.MonthlyEssentialSpend, error) {
	return f.essentialMonthly, f.fail()
}

// --- fakeBudgetRepo ---

type fakeBudgetRepo struct {
	err        error
	budget     *domain.Budget
	categories []*domain.BudgetCategory
	spentMap   map[string]int64
}

func (f *fakeBudgetRepo) fail() error {
	if f.err != nil {
		return f.err
	}
	return nil
}

func (f *fakeBudgetRepo) UpsertBudget(_ context.Context, _ *domain.Budget) error { return f.fail() }
func (f *fakeBudgetRepo) FindBudgetByMonth(_ context.Context, _, _ string) (*domain.Budget, error) {
	return f.budget, f.fail()
}
func (f *fakeBudgetRepo) UpsertBudgetCategories(_ context.Context, _ string, _ []*domain.BudgetCategory) error {
	return f.fail()
}
func (f *fakeBudgetRepo) GetBudgetCategories(_ context.Context, _ string) ([]*domain.BudgetCategory, error) {
	return f.categories, f.fail()
}
func (f *fakeBudgetRepo) GetSpentByCategory(_ context.Context, _, _ string, _, _ time.Time) (map[string]int64, error) {
	return f.spentMap, f.fail()
}

// --- fakeGoalRepo ---

type fakeGoalRepo struct {
	err      error
	goals    []*domain.SavingsGoal
	currents map[string]int64
}

func (f *fakeGoalRepo) fail() error {
	if f.err != nil {
		return f.err
	}
	return nil
}

func (f *fakeGoalRepo) SaveGoal(_ context.Context, _ *domain.SavingsGoal) error { return f.fail() }
func (f *fakeGoalRepo) UpdateGoal(_ context.Context, _ *domain.SavingsGoal) error {
	return f.fail()
}
func (f *fakeGoalRepo) DeleteGoal(_ context.Context, _, _ string) error { return f.fail() }
func (f *fakeGoalRepo) FindGoalByID(_ context.Context, _ string) (*domain.SavingsGoal, error) {
	return nil, f.fail()
}
func (f *fakeGoalRepo) ListGoals(_ context.Context, _ string) ([]*domain.SavingsGoal, error) {
	return f.goals, f.fail()
}
func (f *fakeGoalRepo) SaveContribution(_ context.Context, _ *domain.GoalContribution) error {
	return f.fail()
}
func (f *fakeGoalRepo) FindContributionByIdempotencyKey(_ context.Context, _ string) (*domain.GoalContribution, error) {
	return nil, f.fail()
}
func (f *fakeGoalRepo) ListContributionsByGoal(_ context.Context, _ string) ([]*domain.GoalContribution, error) {
	return nil, f.fail()
}
func (f *fakeGoalRepo) GetCurrentAmountByGoal(_ context.Context, _ string) (int64, error) {
	return 0, f.fail()
}
func (f *fakeGoalRepo) GetCurrentAmountsByGoals(_ context.Context, _ []string) (map[string]int64, error) {
	return f.currents, f.fail()
}

// --- fakeWalletRepo ---

type fakeWalletRepo struct {
	err      error
	balances []*domain.WalletBalance
}

func (f *fakeWalletRepo) fail() error {
	if f.err != nil {
		return f.err
	}
	return nil
}

func (f *fakeWalletRepo) Save(_ context.Context, _ *domain.Wallet) error { return f.fail() }
func (f *fakeWalletRepo) FindByID(_ context.Context, _ string) (*domain.Wallet, error) {
	return nil, f.fail()
}
func (f *fakeWalletRepo) ListByUser(_ context.Context, _ string) ([]*domain.Wallet, error) {
	return nil, f.fail()
}
func (f *fakeWalletRepo) Update(_ context.Context, _ *domain.Wallet) error { return f.fail() }
func (f *fakeWalletRepo) Archive(_ context.Context, _, _ string) error     { return f.fail() }
func (f *fakeWalletRepo) FindDefault(_ context.Context, _ string) (*domain.Wallet, error) {
	return nil, f.fail()
}
func (f *fakeWalletRepo) GetBalancesByUser(_ context.Context, _ string) ([]*domain.WalletBalance, error) {
	return f.balances, f.fail()
}

// --- fakeBillRepo ---

type fakeBillRepo struct {
	err      error
	bills    []*domain.RecurringBill
	payments []*domain.BillPayment
}

func (f *fakeBillRepo) fail() error {
	if f.err != nil {
		return f.err
	}
	return nil
}

func (f *fakeBillRepo) SaveBill(_ context.Context, _ *domain.RecurringBill) error { return f.fail() }
func (f *fakeBillRepo) UpdateBill(_ context.Context, _ *domain.RecurringBill) error {
	return f.fail()
}
func (f *fakeBillRepo) DeleteBill(_ context.Context, _, _ string) error { return f.fail() }
func (f *fakeBillRepo) FindBillByID(_ context.Context, _ string) (*domain.RecurringBill, error) {
	return nil, f.fail()
}
func (f *fakeBillRepo) ListBills(_ context.Context, _ string) ([]*domain.RecurringBill, error) {
	return f.bills, f.fail()
}
func (f *fakeBillRepo) SavePayment(_ context.Context, _ *domain.BillPayment) error { return f.fail() }
func (f *fakeBillRepo) FindPayment(_ context.Context, _, _ string) (*domain.BillPayment, error) {
	return nil, f.fail()
}
func (f *fakeBillRepo) ListPaymentsByBill(_ context.Context, _ string) ([]*domain.BillPayment, error) {
	return nil, f.fail()
}
func (f *fakeBillRepo) ListPaymentsByBills(_ context.Context, _ []string, _, _ time.Time) ([]*domain.BillPayment, error) {
	return f.payments, f.fail()
}
func (f *fakeBillRepo) ListUpcomingBills(_ context.Context, _ string, _ int) ([]*domain.BillWithPayment, error) {
	return nil, f.fail()
}
func (f *fakeBillRepo) FindTransactionByMatch(_ context.Context, _, _ string, _ int64, _ string, _ string) (*domain.Transaction, error) {
	return nil, f.fail()
}
func (f *fakeBillRepo) FindPaymentsByTransaction(_ context.Context, _ string) ([]*domain.BillPayment, error) {
	return nil, f.fail()
}
func (f *fakeBillRepo) DeletePayment(_ context.Context, _ string) error { return f.fail() }

// --- helpers ---

// errRepo is a sentinel used to force repository failures in tests.
var errRepo = errors.New("repository failure")
