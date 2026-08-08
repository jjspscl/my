package domain

import (
	"fmt"
	"strings"
)

// MaxUnclassifiedShare is the maximum fraction of spending that may be
// unclassified before analytics that depend on essential/discretionary
// classification refuse to answer. Locked in the Phase 2 plan: above 20% the
// classification breakdown is not trustworthy enough to report.
const MaxUnclassifiedShare = 0.20

// MinTrendMonths is the minimum sample size before a category trend is
// reported as sufficient. A shorter sample is returned with Sufficient=false
// rather than being presented as a trend.
const MinTrendMonths = 3

// DateRange is a half-open [From, To) range expressed as YYYY-MM-DD strings in
// the user's financial timezone.
type DateRange struct {
	From string
	To   string
}

// ClassificationSpend is one row of the per-currency, per-classification
// expense aggregation. A transaction whose category has no finance_categories
// row (or is classified 'unclassified') lands in ClassificationUnclassified.
type ClassificationSpend struct {
	Currency       string
	Classification Classification
	AmountCents    int64
}

// UnclassifiedSpending is the unclassified-versus-total expense split for one
// currency within a range.
type UnclassifiedSpending struct {
	Currency          string
	UnclassifiedCents int64
	TotalCents        int64
}

// CategorySpend is a category name with its total expense, used to list the
// top unclassified categories when classification is insufficient.
type CategorySpend struct {
	Category    string
	AmountCents int64
}

// MonthlyAmount is one month of a category trend.
type MonthlyAmount struct {
	Month       string // YYYY-MM
	AmountCents int64
}

// MonthlyCashFlow is one month of income/expense/net for one currency.
type MonthlyCashFlow struct {
	Month        string // YYYY-MM
	Currency     string
	IncomeCents  int64
	ExpenseCents int64
	NetCents     int64
}

// CurrencySpending is the spending summary for one currency.
type CurrencySpending struct {
	Currency          string
	TotalExpenseCents int64
	ByClassification  map[Classification]int64
	UnclassifiedCents int64
}

// SpendingSummary is the per-currency expense breakdown by classification.
// UnclassifiedSharePct is the share of total expense in unclassified
// categories; when it exceeds MaxUnclassifiedShare the service refuses with
// ErrInsufficientClassification instead of returning this struct.
type SpendingSummary struct {
	DateRange            DateRange
	Currencies           []CurrencySpending
	UnclassifiedSharePct float64
	Assumptions          []string
}

// CurrencyCashFlow is the cash-flow summary for one currency, with a monthly
// series when the caller asked for one.
type CurrencyCashFlow struct {
	Currency     string
	IncomeCents  int64
	ExpenseCents int64
	NetCents     int64
	Monthly      []MonthlyCashFlow
}

// CashFlowSummary is income/expense/net per currency over a range.
type CashFlowSummary struct {
	DateRange   DateRange
	Currencies  []CurrencyCashFlow
	Assumptions []string
}

// CategoryTrendPoint is one month of a category trend.
type CategoryTrendPoint struct {
	Month       string // YYYY-MM
	AmountCents int64
}

// CategoryTrend is the monthly spending series for one category in one
// currency. Sufficient is false when the sample is shorter than MinTrendMonths;
// the data is still returned, but it must not be presented as a trend.
type CategoryTrend struct {
	Category    string
	Currency    string
	Months      []CategoryTrendPoint
	SampleSize  int
	Sufficient  bool
	Assumptions []string
}

// BudgetHealthCategory is one budgeted category with its spent/remaining.
type BudgetHealthCategory struct {
	Category       string
	AllocatedCents int64
	SpentCents     int64
	RemainingCents int64
}

// BudgetHealth compares a month's plan against actuals. UnbudgetedSpentCents
// is spending in categories that have no allocation for the month; it is
// reported separately, never folded into a budgeted line.
type BudgetHealth struct {
	Month                string
	Currency             string
	HasBudget            bool
	TotalAllocatedCents  int64
	TotalSpentCents      int64
	TotalRemainingCents  int64
	UnbudgetedSpentCents int64
	Categories           []BudgetHealthCategory
	Assumptions          []string
}

// GoalHealthItem is one goal's progress snapshot.
type GoalHealthItem struct {
	ID                   string
	Name                 string
	Currency             string
	TargetAmountCents    int64
	CurrentAmountCents   int64
	RemainingAmountCents int64
	ProgressPercent      int
	Status               GoalStatus
	RequiredMonthlyCents *int64
}

// GoalHealth is the progress snapshot of every goal.
type GoalHealth struct {
	Goals       []GoalHealthItem
	Assumptions []string
}

// SavingsRate is the savings rate for one currency over a range. ZeroIncome
// means the rate is undefined (no income in the range); callers must surface
// that instead of presenting RatePercent.
type SavingsRate struct {
	Currency     string
	IncomeCents  int64
	ExpenseCents int64
	NetCents     int64
	RatePercent  float64
	ZeroIncome   bool
	Assumptions  []string
}

// ErrInsufficientClassification is returned by analytics that depend on
// essential/discretionary classification when more than MaxUnclassifiedShare
// of spending is unclassified. TopUnclassified lists the largest unclassified
// categories so the caller can tell the user what to classify.
type ErrInsufficientClassification struct {
	UnclassifiedSharePct float64
	TopUnclassified      []string
}

func (e *ErrInsufficientClassification) Error() string {
	return fmt.Sprintf(
		"insufficient classification: %.1f%% of spending is unclassified; classify the top categories: %s",
		e.UnclassifiedSharePct, strings.Join(e.TopUnclassified, ", "),
	)
}

// ComputeSavingsRate computes the savings rate as a percentage of income:
// (income - expense) / income * 100. A negative rate means spending exceeded
// income. ZeroIncome is true when income is not positive, in which case the
// rate is undefined and RatePercent is 0.
func ComputeSavingsRate(incomeCents, expenseCents int64) (ratePercent float64, zeroIncome bool) {
	if incomeCents <= 0 {
		return 0, true
	}
	return (float64(incomeCents-expenseCents) / float64(incomeCents)) * 100, false
}
