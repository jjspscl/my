package application

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
	"github.com/jjspscl/my/internal/platform/timeutil"
)

// AnalyticsService is the analytics core: deterministic, SQL GROUP BY
// aggregations in minor units, grouped by currency, with every response
// carrying the assumptions it made. It never sums across currencies and never
// claims a trend from a short sample.
type AnalyticsService struct {
	analyticsRepo domain.AnalyticsRepository
	budgetRepo    domain.BudgetRepository
	goalRepo      domain.GoalRepository
	clock         *timeutil.Clock
}

func NewAnalyticsService(analyticsRepo domain.AnalyticsRepository, budgetRepo domain.BudgetRepository, goalRepo domain.GoalRepository) *AnalyticsService {
	return &AnalyticsService{
		analyticsRepo: analyticsRepo,
		budgetRepo:    budgetRepo,
		goalRepo:      goalRepo,
		clock:         timeutil.New(time.UTC),
	}
}

// WithClock pins the calendar used for month boundaries and trend windows.
func (s *AnalyticsService) WithClock(c *timeutil.Clock) *AnalyticsService {
	s.clock = c
	return s
}

// GetSpendingSummary returns the per-currency expense breakdown by
// classification over [from, to). When more than MaxUnclassifiedShare of
// spending is unclassified it refuses with ErrInsufficientClassification,
// listing the top unclassified categories.
func (s *AnalyticsService) GetSpendingSummary(ctx context.Context, userEmail string, from, to time.Time) (*domain.SpendingSummary, error) {
	spends, err := s.analyticsRepo.GetSpendingByClassification(ctx, userEmail, from, to)
	if err != nil {
		return nil, fmt.Errorf("spending by classification: %w", err)
	}
	unclassified, err := s.analyticsRepo.GetUnclassifiedSpending(ctx, userEmail, from, to)
	if err != nil {
		return nil, fmt.Errorf("unclassified spending: %w", err)
	}

	// Per-currency buckets.
	byCurrency := make(map[string]*domain.CurrencySpending)
	var order []string
	for _, sp := range spends {
		cs, ok := byCurrency[sp.Currency]
		if !ok {
			cs = &domain.CurrencySpending{
				Currency:         sp.Currency,
				ByClassification: make(map[domain.Classification]int64),
			}
			byCurrency[sp.Currency] = cs
			order = append(order, sp.Currency)
		}
		cs.ByClassification[sp.Classification] += sp.AmountCents
		cs.TotalExpenseCents += sp.AmountCents
	}
	for _, u := range unclassified {
		if cs, ok := byCurrency[u.Currency]; ok {
			cs.UnclassifiedCents = u.UnclassifiedCents
		}
	}

	// Overall unclassified share across currencies.
	var totalExpense, totalUnclassified int64
	for _, cs := range byCurrency {
		totalExpense += cs.TotalExpenseCents
		totalUnclassified += cs.UnclassifiedCents
	}
	sharePct := 0.0
	if totalExpense > 0 {
		sharePct = (float64(totalUnclassified) / float64(totalExpense)) * 100
	}

	if sharePct > domain.MaxUnclassifiedShare*100 {
		top, err := s.analyticsRepo.GetTopUnclassifiedCategories(ctx, userEmail, from, to, 5)
		if err != nil {
			return nil, fmt.Errorf("top unclassified categories: %w", err)
		}
		names := make([]string, 0, len(top))
		for _, c := range top {
			names = append(names, c.Category)
		}
		return nil, &domain.ErrInsufficientClassification{
			UnclassifiedSharePct: sharePct,
			TopUnclassified:      names,
		}
	}

	sort.Strings(order)
	currencies := make([]domain.CurrencySpending, 0, len(order))
	for _, cur := range order {
		currencies = append(currencies, *byCurrency[cur])
	}

	return &domain.SpendingSummary{
		DateRange:            dateRangeOf(from, to),
		Currencies:           currencies,
		UnclassifiedSharePct: sharePct,
		Assumptions: []string{
			"unclassified categories are treated as non-essential",
			fmt.Sprintf("unclassified share of spending: %.1f%%", sharePct),
		},
	}, nil
}

// GetCashFlowSummary returns per-currency income/expense/net over [from, to),
// with a per-month series.
func (s *AnalyticsService) GetCashFlowSummary(ctx context.Context, userEmail string, from, to time.Time) (*domain.CashFlowSummary, error) {
	totals, err := s.analyticsRepo.GetCashFlow(ctx, userEmail, from, to)
	if err != nil {
		return nil, fmt.Errorf("cash flow: %w", err)
	}
	monthly, err := s.analyticsRepo.GetMonthlyCashFlow(ctx, userEmail, from, to)
	if err != nil {
		return nil, fmt.Errorf("monthly cash flow: %w", err)
	}

	byCurrency := make(map[string]*domain.CurrencyCashFlow)
	var order []string
	for _, t := range totals {
		cc := &domain.CurrencyCashFlow{
			Currency:     t.Currency,
			IncomeCents:  t.IncomeCents,
			ExpenseCents: t.ExpenseCents,
			NetCents:     t.TotalCents,
		}
		byCurrency[t.Currency] = cc
		order = append(order, t.Currency)
	}
	for _, m := range monthly {
		if cc, ok := byCurrency[m.Currency]; ok {
			cc.Monthly = append(cc.Monthly, m)
		}
	}

	sort.Strings(order)
	currencies := make([]domain.CurrencyCashFlow, 0, len(order))
	for _, cur := range order {
		currencies = append(currencies, *byCurrency[cur])
	}

	return &domain.CashFlowSummary{
		DateRange:  dateRangeOf(from, to),
		Currencies: currencies,
		Assumptions: []string{
			"amounts are grouped by currency; no cross-currency conversion is applied",
		},
	}, nil
}

// GetCategoryTrend returns the monthly spending series for one category in one
// currency over the last `months` months (including the current month). The
// series is zero-filled for months with no spending. Sufficient is false when
// months < MinTrendMonths; the data is returned but must not be presented as a
// trend.
func (s *AnalyticsService) GetCategoryTrend(ctx context.Context, userEmail, category, currency string, months int) (*domain.CategoryTrend, error) {
	if category == "" {
		return nil, fmt.Errorf("category is required")
	}
	if currency == "" {
		return nil, fmt.Errorf("currency is required")
	}
	if months < 1 || months > 24 {
		return nil, fmt.Errorf("months must be between 1 and 24")
	}

	// Window: from the start of (current month - months + 1) to the start of
	// the month after the current month (half-open).
	now := s.clock.Now()
	currentMonth := now.Format("2006-01")
	fromMonth := addMonths(currentMonth, -(months - 1))
	from, err := s.clock.ParseDate(fromMonth + "-01")
	if err != nil {
		return nil, fmt.Errorf("trend from: %w", err)
	}
	_, to, err := s.clock.MonthRange(addMonths(currentMonth, 1))
	if err != nil {
		return nil, fmt.Errorf("trend to: %w", err)
	}

	amounts, err := s.analyticsRepo.GetCategoryMonthlySpend(ctx, userEmail, category, currency, from, to)
	if err != nil {
		return nil, fmt.Errorf("category monthly spend: %w", err)
	}

	// Zero-fill every month in the window so the series is continuous.
	byMonth := make(map[string]int64, len(amounts))
	for _, a := range amounts {
		byMonth[a.Month] = a.AmountCents
	}
	points := make([]domain.CategoryTrendPoint, 0, months)
	for i := 0; i < months; i++ {
		m := addMonths(fromMonth, i)
		points = append(points, domain.CategoryTrendPoint{Month: m, AmountCents: byMonth[m]})
	}

	return &domain.CategoryTrend{
		Category:   category,
		Currency:   currency,
		Months:     points,
		SampleSize: months,
		Sufficient: months >= domain.MinTrendMonths,
		Assumptions: []string{
			fmt.Sprintf("trend window: %d months ending %s", months, currentMonth),
			fmt.Sprintf("a trend requires at least %d months of data; shorter samples are not presented as a trend", domain.MinTrendMonths),
		},
	}, nil
}

// GetBudgetHealth compares a month's plan against actuals. Unbudgeted spending
// is reported separately, never folded into a budgeted line.
func (s *AnalyticsService) GetBudgetHealth(ctx context.Context, userEmail, month string) (*domain.BudgetHealth, error) {
	budget, err := s.budgetRepo.FindBudgetByMonth(ctx, userEmail, month)
	if err != nil {
		return nil, fmt.Errorf("find budget: %w", err)
	}

	health := &domain.BudgetHealth{
		Month:      month,
		Categories: []domain.BudgetHealthCategory{},
		Assumptions: []string{
			"unbudgeted spending is reported separately from budgeted lines",
		},
	}
	if budget == nil {
		health.Assumptions = append(health.Assumptions, "no budget set for this month")
		return health, nil
	}

	health.HasBudget = true
	health.Currency = budget.Currency

	categories, err := s.budgetRepo.GetBudgetCategories(ctx, budget.ID)
	if err != nil {
		return nil, fmt.Errorf("get budget categories: %w", err)
	}

	from, to, err := s.clock.MonthRange(month)
	if err != nil {
		return nil, fmt.Errorf("month range: %w", err)
	}

	spentMap, err := s.budgetRepo.GetSpentByCategory(ctx, userEmail, budget.Currency, from, to)
	if err != nil {
		return nil, fmt.Errorf("get spent by category: %w", err)
	}

	for _, cat := range categories {
		spent := spentMap[cat.Category]
		health.Categories = append(health.Categories, domain.BudgetHealthCategory{
			Category:       cat.Category,
			AllocatedCents: cat.AllocatedCents,
			SpentCents:     spent,
			RemainingCents: cat.AllocatedCents - spent,
		})
		health.TotalAllocatedCents += cat.AllocatedCents
		health.TotalSpentCents += spent
	}
	health.TotalRemainingCents = health.TotalAllocatedCents - health.TotalSpentCents

	unbudgeted, err := s.analyticsRepo.GetUnbudgetedSpend(ctx, userEmail, budget.Currency, month, from, to)
	if err != nil {
		return nil, fmt.Errorf("get unbudgeted spend: %w", err)
	}
	health.UnbudgetedSpentCents = unbudgeted

	return health, nil
}

// GetGoalHealth returns the progress snapshot of every goal.
func (s *AnalyticsService) GetGoalHealth(ctx context.Context, userEmail string) (*domain.GoalHealth, error) {
	goals, err := s.goalRepo.ListGoals(ctx, userEmail)
	if err != nil {
		return nil, fmt.Errorf("list goals: %w", err)
	}

	ids := make([]string, 0, len(goals))
	for _, g := range goals {
		ids = append(ids, g.ID)
	}
	currents, err := s.goalRepo.GetCurrentAmountsByGoals(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get current amounts: %w", err)
	}

	now := s.clock.Now()
	items := make([]domain.GoalHealthItem, 0, len(goals))
	for _, goal := range goals {
		summary := domain.ComputeGoalSummary(goal, currents[goal.ID], now)
		items = append(items, domain.GoalHealthItem{
			ID:                   goal.ID,
			Name:                 goal.Name,
			Currency:             goal.Currency,
			TargetAmountCents:    goal.TargetAmountCents,
			CurrentAmountCents:   summary.CurrentAmountCents,
			RemainingAmountCents: summary.RemainingAmountCents,
			ProgressPercent:      summary.ProgressPercent,
			Status:               summary.Status,
			RequiredMonthlyCents: summary.RequiredMonthlyCents,
		})
	}

	return &domain.GoalHealth{
		Goals:       items,
		Assumptions: []string{"all goals are included; progress is computed from contributions"},
	}, nil
}

// GetSavingsRate returns the per-currency savings rate over [from, to).
// Definition: (income - expense) / income. A negative rate means spending
// exceeded income. ZeroIncome is true when a currency had no income, in which
// case the rate is undefined.
func (s *AnalyticsService) GetSavingsRate(ctx context.Context, userEmail string, from, to time.Time) ([]domain.SavingsRate, error) {
	totals, err := s.analyticsRepo.GetCashFlow(ctx, userEmail, from, to)
	if err != nil {
		return nil, fmt.Errorf("cash flow: %w", err)
	}

	rates := make([]domain.SavingsRate, 0, len(totals))
	for _, t := range totals {
		ratePct, zero := domain.ComputeSavingsRate(t.IncomeCents, t.ExpenseCents)
		rates = append(rates, domain.SavingsRate{
			Currency:     t.Currency,
			IncomeCents:  t.IncomeCents,
			ExpenseCents: t.ExpenseCents,
			NetCents:     t.TotalCents,
			RatePercent:  ratePct,
			ZeroIncome:   zero,
			Assumptions: []string{
				"savings rate = (income - expense) / income",
				"a negative rate means spending exceeded income",
			},
		})
	}

	return rates, nil
}

func dateRangeOf(from, to time.Time) domain.DateRange {
	return domain.DateRange{From: from.Format("2006-01-02"), To: to.Format("2006-01-02")}
}

// addMonths shifts a YYYY-MM string by n months (n may be negative).
func addMonths(month string, n int) string {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return month
	}
	return t.AddDate(0, n, 0).Format("2006-01")
}
