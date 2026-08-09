package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
	"github.com/jjspscl/my/internal/platform/timeutil"
)

const analyticsTestUser = "user@example.com"

type mockAnalyticsRepo struct {
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
}

func (m *mockAnalyticsRepo) GetCashFlow(_ context.Context, _ string, _, _ time.Time) ([]domain.CurrencyTotal, error) {
	return m.cashFlow, nil
}
func (m *mockAnalyticsRepo) GetMonthlyCashFlow(_ context.Context, _ string, _, _ time.Time) ([]domain.MonthlyCashFlow, error) {
	return m.monthlyFlow, nil
}
func (m *mockAnalyticsRepo) GetSpendingByClassification(_ context.Context, _ string, _, _ time.Time) ([]domain.ClassificationSpend, error) {
	return m.classification, nil
}
func (m *mockAnalyticsRepo) GetUnclassifiedSpending(_ context.Context, _ string, _, _ time.Time) ([]domain.UnclassifiedSpending, error) {
	return m.unclassified, nil
}
func (m *mockAnalyticsRepo) GetTopUnclassifiedCategories(_ context.Context, _ string, _, _ time.Time, _ int) ([]domain.CategorySpend, error) {
	return m.topUnclassified, nil
}
func (m *mockAnalyticsRepo) GetCategoryMonthlySpend(_ context.Context, _, _, _ string, _, _ time.Time) ([]domain.MonthlyAmount, error) {
	return m.categoryMonthly, nil
}
func (m *mockAnalyticsRepo) GetUnbudgetedSpend(_ context.Context, _, _, _ string, _, _ time.Time) (int64, error) {
	return m.unbudgeted, nil
}
func (m *mockAnalyticsRepo) GetCategoryMonthlySpendAll(_ context.Context, _ string, _, _ time.Time) ([]domain.CategoryMonthlySpend, error) {
	return m.categoryMonthlyAll, nil
}
func (m *mockAnalyticsRepo) GetExpenseAmounts(_ context.Context, _ string, _, _ time.Time) ([]domain.ExpenseAmount, error) {
	return m.expenseAmounts, nil
}
func (m *mockAnalyticsRepo) GetBillReconciliation(_ context.Context, _ string, _, _ time.Time) ([]domain.BillReconciliationRow, error) {
	return m.billReconciliation, nil
}

func newAnalyticsService(repo *mockAnalyticsRepo, budget *mockBudgetRepo, goal *mockGoalRepo) *AnalyticsService {
	loc, _ := time.LoadLocation("Asia/Manila")
	return NewAnalyticsService(repo, budget, goal).WithClock(timeutil.New(loc))
}

func TestGetSpendingSummaryClassifies(t *testing.T) {
	repo := &mockAnalyticsRepo{
		classification: []domain.ClassificationSpend{
			{Currency: "PHP", Classification: domain.ClassificationNeeds, AmountCents: 6000},
			{Currency: "PHP", Classification: domain.ClassificationWants, AmountCents: 3000},
		},
		unclassified: []domain.UnclassifiedSpending{
			{Currency: "PHP", UnclassifiedCents: 0, TotalCents: 9000},
		},
	}
	svc := newAnalyticsService(repo, newMockBudgetRepo(), &mockGoalRepo{})

	summary, err := svc.GetSpendingSummary(context.Background(), analyticsTestUser, time.Now().AddDate(0, -1, 0), time.Now())
	if err != nil {
		t.Fatalf("GetSpendingSummary: %v", err)
	}
	if len(summary.Currencies) != 1 {
		t.Fatalf("expected 1 currency, got %d", len(summary.Currencies))
	}
	cs := summary.Currencies[0]
	if cs.TotalExpenseCents != 9000 {
		t.Errorf("total = %d, want 9000", cs.TotalExpenseCents)
	}
	if cs.ByClassification[domain.ClassificationNeeds] != 6000 {
		t.Errorf("needs = %d, want 6000", cs.ByClassification[domain.ClassificationNeeds])
	}
	if summary.UnclassifiedSharePct != 0 {
		t.Errorf("unclassified share = %v, want 0", summary.UnclassifiedSharePct)
	}
}

func TestGetSpendingSummaryRefusesAboveThreshold(t *testing.T) {
	repo := &mockAnalyticsRepo{
		classification: []domain.ClassificationSpend{
			{Currency: "PHP", Classification: domain.ClassificationUnclassified, AmountCents: 8000},
			{Currency: "PHP", Classification: domain.ClassificationNeeds, AmountCents: 2000},
		},
		unclassified: []domain.UnclassifiedSpending{
			{Currency: "PHP", UnclassifiedCents: 8000, TotalCents: 10000},
		},
		topUnclassified: []domain.CategorySpend{
			{Category: "Food", AmountCents: 5000},
			{Category: "Shopping", AmountCents: 3000},
		},
	}
	svc := newAnalyticsService(repo, newMockBudgetRepo(), &mockGoalRepo{})

	_, err := svc.GetSpendingSummary(context.Background(), analyticsTestUser, time.Now().AddDate(0, -1, 0), time.Now())
	var insufficient *domain.ErrInsufficientClassification
	if !errors.As(err, &insufficient) {
		t.Fatalf("expected ErrInsufficientClassification, got %v", err)
	}
	if len(insufficient.TopUnclassified) != 2 {
		t.Errorf("expected 2 top unclassified categories, got %d", len(insufficient.TopUnclassified))
	}
	if insufficient.UnclassifiedSharePct != 80 {
		t.Errorf("share = %v, want 80", insufficient.UnclassifiedSharePct)
	}
}

func TestGetSpendingSummaryAllowsBelowThreshold(t *testing.T) {
	repo := &mockAnalyticsRepo{
		classification: []domain.ClassificationSpend{
			{Currency: "PHP", Classification: domain.ClassificationUnclassified, AmountCents: 1000},
			{Currency: "PHP", Classification: domain.ClassificationNeeds, AmountCents: 9000},
		},
		unclassified: []domain.UnclassifiedSpending{
			{Currency: "PHP", UnclassifiedCents: 1000, TotalCents: 10000},
		},
	}
	svc := newAnalyticsService(repo, newMockBudgetRepo(), &mockGoalRepo{})

	summary, err := svc.GetSpendingSummary(context.Background(), analyticsTestUser, time.Now().AddDate(0, -1, 0), time.Now())
	if err != nil {
		t.Fatalf("GetSpendingSummary: %v", err)
	}
	if summary.UnclassifiedSharePct != 10 {
		t.Errorf("share = %v, want 10", summary.UnclassifiedSharePct)
	}
	if summary.Currencies[0].UnclassifiedCents != 1000 {
		t.Errorf("unclassified cents = %d, want 1000", summary.Currencies[0].UnclassifiedCents)
	}
}

func TestGetCashFlowSummaryGroupsByCurrency(t *testing.T) {
	repo := &mockAnalyticsRepo{
		cashFlow: []domain.CurrencyTotal{
			{Currency: "PHP", IncomeCents: 50000, ExpenseCents: 20000, TotalCents: 30000},
			{Currency: "USD", IncomeCents: 1000, ExpenseCents: 400, TotalCents: 600},
		},
		monthlyFlow: []domain.MonthlyCashFlow{
			{Month: "2026-07", Currency: "PHP", IncomeCents: 50000, ExpenseCents: 20000, NetCents: 30000},
		},
	}
	svc := newAnalyticsService(repo, newMockBudgetRepo(), &mockGoalRepo{})

	summary, err := svc.GetCashFlowSummary(context.Background(), analyticsTestUser, time.Now().AddDate(0, -1, 0), time.Now())
	if err != nil {
		t.Fatalf("GetCashFlowSummary: %v", err)
	}
	if len(summary.Currencies) != 2 {
		t.Fatalf("expected 2 currencies, got %d", len(summary.Currencies))
	}
	// Sorted by currency: PHP, USD.
	if summary.Currencies[0].Currency != "PHP" || summary.Currencies[1].Currency != "USD" {
		t.Errorf("unexpected currency order: %s, %s", summary.Currencies[0].Currency, summary.Currencies[1].Currency)
	}
	if len(summary.Currencies[0].Monthly) != 1 {
		t.Errorf("expected 1 monthly row for PHP, got %d", len(summary.Currencies[0].Monthly))
	}
}

func TestGetCategoryTrendSufficientAndZeroFilled(t *testing.T) {
	repo := &mockAnalyticsRepo{
		categoryMonthly: []domain.MonthlyAmount{
			// Only one month has data; the rest must be zero-filled.
			{Month: currentMonthOffset(-1), AmountCents: 5000},
		},
	}
	svc := newAnalyticsService(repo, newMockBudgetRepo(), &mockGoalRepo{})

	trend, err := svc.GetCategoryTrend(context.Background(), analyticsTestUser, "Food", "PHP", 6)
	if err != nil {
		t.Fatalf("GetCategoryTrend: %v", err)
	}
	if len(trend.Months) != 6 {
		t.Errorf("expected 6 months, got %d", len(trend.Months))
	}
	if !trend.Sufficient {
		t.Error("6 months should be sufficient")
	}
	found := false
	for _, m := range trend.Months {
		if m.AmountCents == 5000 {
			found = true
		}
	}
	if !found {
		t.Error("expected the one non-zero month to appear in the series")
	}
}

func TestGetCategoryTrendInsufficientSample(t *testing.T) {
	svc := newAnalyticsService(&mockAnalyticsRepo{}, newMockBudgetRepo(), &mockGoalRepo{})

	trend, err := svc.GetCategoryTrend(context.Background(), analyticsTestUser, "Food", "PHP", 1)
	if err != nil {
		t.Fatalf("GetCategoryTrend: %v", err)
	}
	if trend.Sufficient {
		t.Error("1 month must not be sufficient for a trend")
	}
	if trend.SampleSize != 1 {
		t.Errorf("sample size = %d, want 1", trend.SampleSize)
	}
}

func TestGetCategoryTrendRejectsBadInput(t *testing.T) {
	svc := newAnalyticsService(&mockAnalyticsRepo{}, newMockBudgetRepo(), &mockGoalRepo{})

	if _, err := svc.GetCategoryTrend(context.Background(), analyticsTestUser, "", "PHP", 6); err == nil {
		t.Error("expected error for empty category")
	}
	if _, err := svc.GetCategoryTrend(context.Background(), analyticsTestUser, "Food", "", 6); err == nil {
		t.Error("expected error for empty currency")
	}
	if _, err := svc.GetCategoryTrend(context.Background(), analyticsTestUser, "Food", "PHP", 0); err == nil {
		t.Error("expected error for months < 1")
	}
	if _, err := svc.GetCategoryTrend(context.Background(), analyticsTestUser, "Food", "PHP", 25); err == nil {
		t.Error("expected error for months > 24")
	}
}

func TestGetBudgetHealthWithBudget(t *testing.T) {
	budget := newMockBudgetRepo()
	budget.budgets["user@example.com:2026-07"] = &domain.Budget{ID: "b1", UserEmail: "user@example.com", Month: "2026-07", Currency: "PHP"}
	budget.categories["b1"] = []*domain.BudgetCategory{
		{Category: "Food", AllocatedCents: 10000},
		{Category: "Transport", AllocatedCents: 5000},
	}
	budget.spentMap = map[string]int64{"Food": 4000, "Transport": 7000}

	repo := &mockAnalyticsRepo{unbudgeted: 1500}
	svc := newAnalyticsService(repo, budget, &mockGoalRepo{})

	health, err := svc.GetBudgetHealth(context.Background(), analyticsTestUser, "2026-07")
	if err != nil {
		t.Fatalf("GetBudgetHealth: %v", err)
	}
	if !health.HasBudget {
		t.Fatal("expected budget to exist")
	}
	if health.Currency != "PHP" {
		t.Errorf("currency = %s, want PHP", health.Currency)
	}
	if health.TotalAllocatedCents != 15000 || health.TotalSpentCents != 11000 {
		t.Errorf("totals allocated=%d spent=%d, want 15000/11000", health.TotalAllocatedCents, health.TotalSpentCents)
	}
	if health.TotalRemainingCents != 4000 {
		t.Errorf("remaining = %d, want 4000", health.TotalRemainingCents)
	}
	if health.UnbudgetedSpentCents != 1500 {
		t.Errorf("unbudgeted = %d, want 1500", health.UnbudgetedSpentCents)
	}
	if len(health.Categories) != 2 {
		t.Errorf("expected 2 categories, got %d", len(health.Categories))
	}
}

func TestGetBudgetHealthNoBudget(t *testing.T) {
	svc := newAnalyticsService(&mockAnalyticsRepo{}, newMockBudgetRepo(), &mockGoalRepo{})

	health, err := svc.GetBudgetHealth(context.Background(), analyticsTestUser, "2026-07")
	if err != nil {
		t.Fatalf("GetBudgetHealth: %v", err)
	}
	if health.HasBudget {
		t.Error("expected no budget")
	}
	if len(health.Categories) != 0 {
		t.Errorf("expected no categories, got %d", len(health.Categories))
	}
}

func TestGetGoalHealth(t *testing.T) {
	goals := &mockGoalRepo{
		goals:         map[string]*domain.SavingsGoal{},
		contributions: map[string][]*domain.GoalContribution{},
	}
	now := time.Now().UTC()
	goals.goals["g1"] = &domain.SavingsGoal{
		ID: "g1", UserEmail: analyticsTestUser, Name: "Emergency fund",
		TargetAmountCents: 100000, Currency: "PHP",
		TargetWalletID: "w1", CreatedAt: now, UpdatedAt: now,
	}
	goals.contributions["g1"] = []*domain.GoalContribution{
		{ID: "c1", GoalID: "g1", AmountCents: 25000, ContributedAt: now},
	}

	svc := newAnalyticsService(&mockAnalyticsRepo{}, newMockBudgetRepo(), goals)
	health, err := svc.GetGoalHealth(context.Background(), analyticsTestUser)
	if err != nil {
		t.Fatalf("GetGoalHealth: %v", err)
	}
	if len(health.Goals) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(health.Goals))
	}
	g := health.Goals[0]
	if g.CurrentAmountCents != 25000 {
		t.Errorf("current = %d, want 25000", g.CurrentAmountCents)
	}
	if g.ProgressPercent != 25 {
		t.Errorf("progress = %d, want 25", g.ProgressPercent)
	}
	if g.Status != domain.GoalInProgress {
		t.Errorf("status = %s, want in_progress", g.Status)
	}
}

func TestGetSavingsRate(t *testing.T) {
	repo := &mockAnalyticsRepo{
		cashFlow: []domain.CurrencyTotal{
			{Currency: "PHP", IncomeCents: 10000, ExpenseCents: 6000, TotalCents: 4000},
			{Currency: "USD", IncomeCents: 0, ExpenseCents: 500, TotalCents: -500},
		},
	}
	svc := newAnalyticsService(repo, newMockBudgetRepo(), &mockGoalRepo{})

	rates, err := svc.GetSavingsRate(context.Background(), analyticsTestUser, time.Now().AddDate(0, -1, 0), time.Now())
	if err != nil {
		t.Fatalf("GetSavingsRate: %v", err)
	}
	if len(rates) != 2 {
		t.Fatalf("expected 2 rates, got %d", len(rates))
	}
	for _, rate := range rates {
		if rate.Currency == "PHP" {
			if rate.RatePercent != 40 {
				t.Errorf("PHP rate = %v, want 40", rate.RatePercent)
			}
			if rate.ZeroIncome {
				t.Error("PHP should not be zero income")
			}
		}
		if rate.Currency == "USD" {
			if !rate.ZeroIncome {
				t.Error("USD should be zero income")
			}
			if rate.RatePercent != 0 {
				t.Errorf("USD rate = %v, want 0 (undefined)", rate.RatePercent)
			}
		}
	}
}

// currentMonthOffset returns the YYYY-MM month n months before the current
// month, resolved in Manila (the test clock location) so it matches the
// service's window at month boundaries.
func currentMonthOffset(n int) string {
	loc, _ := time.LoadLocation("Asia/Manila")
	return time.Now().In(loc).AddDate(0, n, 0).Format("2006-01")
}
