package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/jjspscl/my/internal/platform/bootstrap"
	"github.com/jjspscl/my/internal/platform/timeutil"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Analytics tools wrap the read-only AnalyticsService and
// DerivedAnalyticsService surfaces exposed on bootstrap.App. Every tool is
// annotated read-only + idempotent. Date ranges default to the current month
// in the configured location, mirroring the HTTP analytics handlers.

type analyticsRangeInput struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type categoryTrendInput struct {
	Category string `json:"category"`
	Currency string `json:"currency"`
	Months   int    `json:"months,omitempty"`
}

type monthInput struct {
	Month string `json:"month"`
}

type currencyMonthsInput struct {
	Currency string `json:"currency"`
	Months   int    `json:"months,omitempty"`
}

type emergencyFundInput struct {
	Currency     string `json:"currency"`
	TargetMonths int    `json:"target_months,omitempty"`
}

type affordabilityInput struct {
	Currency    string `json:"currency"`
	AmountCents int64  `json:"amount_cents"`
}

func registerAnalyticsReadTools(server *mcpsdk.Server, app *bootstrap.App) {
	readOnly := &mcpsdk.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true}

	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_spending_summary", Description: "Per-currency expense breakdown by classification (needs, wants, unclassified) over a date range; defaults to the current month. Refuses when too much spending is unclassified.", Annotations: readOnly}, func(ctx context.Context, in analyticsRangeInput) (any, error) {
		from, to, err := resolveRange(app, in.From, in.To)
		if err != nil {
			return nil, err
		}
		return app.Analytics.GetSpendingSummary(ctx, app.Cfg.UserEmail, from, to)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_cash_flow_summary", Description: "Per-currency income, expense, and net over a date range with a monthly series; defaults to the current month.", Annotations: readOnly}, func(ctx context.Context, in analyticsRangeInput) (any, error) {
		from, to, err := resolveRange(app, in.From, in.To)
		if err != nil {
			return nil, err
		}
		return app.Analytics.GetCashFlowSummary(ctx, app.Cfg.UserEmail, from, to)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_category_trend", Description: "Monthly spending series for one category in one currency over the last months (default 6, max 24). Shorter samples are returned but flagged as insufficient for a trend.", Annotations: readOnly}, func(ctx context.Context, in categoryTrendInput) (any, error) {
		months := in.Months
		if months == 0 {
			months = 6
		}
		return app.Analytics.GetCategoryTrend(ctx, app.Cfg.UserEmail, in.Category, in.Currency, months)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_budget_health", Description: "Compare a month's budget plan against actuals, including unbudgeted spending; defaults to the current month.", Annotations: readOnly}, func(ctx context.Context, in monthInput) (any, error) {
		month := in.Month
		if month == "" {
			month = timeutil.New(app.Cfg.Location).CurrentMonth()
		}
		return app.Analytics.GetBudgetHealth(ctx, app.Cfg.UserEmail, month)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_goal_health", Description: "Progress snapshot of every savings goal: current, remaining, progress percent, and required monthly contribution.", Annotations: readOnly}, func(ctx context.Context, _ emptyInput) (any, error) {
		return app.Analytics.GetGoalHealth(ctx, app.Cfg.UserEmail)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_savings_rate", Description: "Per-currency savings rate (income minus expense over income) over a date range; defaults to the current month.", Annotations: readOnly}, func(ctx context.Context, in analyticsRangeInput) (any, error) {
		from, to, err := resolveRange(app, in.From, in.To)
		if err != nil {
			return nil, err
		}
		return app.Analytics.GetSavingsRate(ctx, app.Cfg.UserEmail, from, to)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_anomalies", Description: "Flag unusual monthly spending per category in one currency over the last months (default 6, up to 24) using a Hampel filter.", Annotations: readOnly}, func(ctx context.Context, in currencyMonthsInput) (any, error) {
		months := in.Months
		if months == 0 {
			months = 6
		}
		return app.DerivedAnalytics.GetMonthlyAnomalies(ctx, app.Cfg.UserEmail, in.Currency, months)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_recurring_charges", Description: "Recurring-charge summary in one currency over the last months (default 6, up to 24), classified as tracked, untracked, or amount_changed against explicit bills.", Annotations: readOnly}, func(ctx context.Context, in currencyMonthsInput) (any, error) {
		months := in.Months
		if months == 0 {
			months = 6
		}
		return app.DerivedAnalytics.GetRecurringCharges(ctx, app.Cfg.UserEmail, in.Currency, months)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_bill_reconciliation", Description: "Compare each bill's expected amount against what was actually paid in a month (YYYY-MM), including paid occurrences without a linked transaction.", Annotations: readOnly}, func(ctx context.Context, in monthInput) (any, error) {
		return app.DerivedAnalytics.GetBillReconciliation(ctx, app.Cfg.UserEmail, in.Month)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_emergency_fund", Description: "Liquid balance against a target of months of essential spending (default 3-6, override with target_months 1-12) in one currency.", Annotations: readOnly}, func(ctx context.Context, in emergencyFundInput) (any, error) {
		return app.DerivedAnalytics.GetEmergencyFund(ctx, app.Cfg.UserEmail, in.Currency, in.TargetMonths)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_affordability", Description: "Model a prospective purchase in one currency: runway before and after, given essential spending and unpaid bills due in the next 30 days. Returns a model, never a yes/no.", Annotations: readOnly}, func(ctx context.Context, in affordabilityInput) (any, error) {
		return app.DerivedAnalytics.GetAffordability(ctx, app.Cfg.UserEmail, in.Currency, in.AmountCents)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_monthly_digest", Description: "Compose the monthly summary for a month (YYYY-MM): cash flow, spending breakdown, savings rate, recurring charges, anomalies, and emergency fund. Sections that cannot be computed are omitted with the reason.", Annotations: readOnly}, func(ctx context.Context, in monthInput) (any, error) {
		return app.DerivedAnalytics.GetMonthlyDigest(ctx, app.Cfg.UserEmail, in.Month)
	})
}

// resolveRange returns the requested [from, to) range, defaulting to the
// current month in the configured location when both are omitted. Providing
// only one bound is an error.
func resolveRange(app *bootstrap.App, from, to time.Time) (time.Time, time.Time, error) {
	if from.IsZero() && to.IsZero() {
		clock := timeutil.New(app.Cfg.Location)
		return clock.MonthRange(clock.CurrentMonth())
	}
	if from.IsZero() || to.IsZero() {
		return time.Time{}, time.Time{}, fmt.Errorf("from and to must both be provided or both omitted")
	}
	return from, to, nil
}
