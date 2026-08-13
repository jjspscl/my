package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
)

// --- /spending ---

func TestSpendingSummaryMapsClassification(t *testing.T) {
	analytics := &fakeAnalyticsRepo{
		classification: []domain.ClassificationSpend{
			{Currency: "PHP", Classification: domain.ClassificationNeeds, AmountCents: 6000},
			{Currency: "PHP", Classification: domain.ClassificationWants, AmountCents: 3000},
		},
		unclassified: []domain.UnclassifiedSpending{
			{Currency: "PHP", UnclassifiedCents: 0, TotalCents: 9000},
		},
	}
	analyticsSvc, derived := newTestServices(analytics, &fakeBudgetRepo{}, &fakeGoalRepo{}, &fakeWalletRepo{}, &fakeBillRepo{})
	router := newAnalyticsRouter(analyticsSvc, derived)

	rec := doAnalyticsRequest(t, router, "/analytics/spending?from=2026-01-01&to=2026-02-01")

	var out struct {
		DateRange struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"dateRange"`
		Currencies []struct {
			Currency          string           `json:"currency"`
			TotalExpenseCents int64            `json:"totalExpenseCents"`
			ByClassification  map[string]int64 `json:"byClassification"`
			UnclassifiedCents int64            `json:"unclassifiedCents"`
		} `json:"currencies"`
		UnclassifiedSharePct float64  `json:"unclassifiedSharePct"`
		Assumptions          []string `json:"assumptions"`
	}
	decodeData(t, rec, &out)

	if out.DateRange.From != "2026-01-01" || out.DateRange.To != "2026-02-01" {
		t.Errorf("dateRange = %+v, want 2026-01-01..2026-02-01", out.DateRange)
	}
	if len(out.Currencies) != 1 {
		t.Fatalf("currencies = %d, want 1", len(out.Currencies))
	}
	c := out.Currencies[0]
	if c.Currency != "PHP" || c.TotalExpenseCents != 9000 {
		t.Errorf("currency = %s total = %d, want PHP 9000", c.Currency, c.TotalExpenseCents)
	}
	if c.ByClassification["needs"] != 6000 || c.ByClassification["wants"] != 3000 {
		t.Errorf("byClassification = %v, want needs=6000 wants=3000", c.ByClassification)
	}
	if out.UnclassifiedSharePct != 0 {
		t.Errorf("unclassifiedSharePct = %v, want 0", out.UnclassifiedSharePct)
	}
}

func TestSpendingSummary422OnInsufficientClassification(t *testing.T) {
	analytics := &fakeAnalyticsRepo{
		classification: []domain.ClassificationSpend{
			{Currency: "PHP", Classification: domain.ClassificationUnclassified, AmountCents: 8000},
			{Currency: "PHP", Classification: domain.ClassificationNeeds, AmountCents: 2000},
		},
		unclassified: []domain.UnclassifiedSpending{
			{Currency: "PHP", UnclassifiedCents: 8000, TotalCents: 10000},
		},
		topUnclassified: []domain.CategorySpend{
			{Category: "Food", AmountCents: 5000},
		},
	}
	analyticsSvc, derived := newTestServices(analytics, &fakeBudgetRepo{}, &fakeGoalRepo{}, &fakeWalletRepo{}, &fakeBillRepo{})
	router := newAnalyticsRouter(analyticsSvc, derived)

	rec := doAnalyticsRequest(t, router, "/analytics/spending?from=2026-01-01&to=2026-02-01")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body: %s", rec.Code, rec.Body.String())
	}
	var errBody struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
		t.Fatalf("unmarshal error body: %v", err)
	}
	if !strings.Contains(errBody.Error, "insufficient classification") || !strings.Contains(errBody.Error, "Food") {
		t.Errorf("error = %q, want insufficient classification naming Food", errBody.Error)
	}
}

// --- /cash-flow ---

func TestCashFlowSummaryMapsMonthlySeries(t *testing.T) {
	analytics := &fakeAnalyticsRepo{
		cashFlow: []domain.CurrencyTotal{
			{Currency: "PHP", TotalCents: 3000, ExpenseCents: 7000, IncomeCents: 10000},
		},
		monthlyFlow: []domain.MonthlyCashFlow{
			{Month: "2026-01", Currency: "PHP", IncomeCents: 10000, ExpenseCents: 7000, NetCents: 3000},
		},
	}
	analyticsSvc, derived := newTestServices(analytics, &fakeBudgetRepo{}, &fakeGoalRepo{}, &fakeWalletRepo{}, &fakeBillRepo{})
	router := newAnalyticsRouter(analyticsSvc, derived)

	rec := doAnalyticsRequest(t, router, "/analytics/cash-flow?from=2026-01-01&to=2026-02-01")
	var out struct {
		Currencies []struct {
			Currency     string `json:"currency"`
			IncomeCents  int64  `json:"incomeCents"`
			ExpenseCents int64  `json:"expenseCents"`
			NetCents     int64  `json:"netCents"`
			Monthly      []struct {
				Month        string `json:"month"`
				Currency     string `json:"currency"`
				IncomeCents  int64  `json:"incomeCents"`
				ExpenseCents int64  `json:"expenseCents"`
				NetCents     int64  `json:"netCents"`
			} `json:"monthly"`
		} `json:"currencies"`
	}
	decodeData(t, rec, &out)

	if len(out.Currencies) != 1 {
		t.Fatalf("currencies = %d, want 1", len(out.Currencies))
	}
	c := out.Currencies[0]
	if c.Currency != "PHP" || c.IncomeCents != 10000 || c.ExpenseCents != 7000 || c.NetCents != 3000 {
		t.Errorf("currency summary = %+v, want PHP income=10000 expense=7000 net=3000", c)
	}
	if len(c.Monthly) != 1 {
		t.Fatalf("monthly = %d, want 1", len(c.Monthly))
	}
	m := c.Monthly[0]
	if m.Month != "2026-01" || m.NetCents != 3000 {
		t.Errorf("monthly row = %+v, want 2026-01 net=3000", m)
	}
}

// --- /category-trend ---

func TestCategoryTrendZeroFillsAndSufficient(t *testing.T) {
	// The service computes the window from the real clock, so seed the fake
	// with the current month to land inside it.
	currentMonth := time.Now().UTC().Format("2006-01")
	analytics := &fakeAnalyticsRepo{
		categoryMonthly: []domain.MonthlyAmount{
			{Month: currentMonth, AmountCents: 5000},
		},
	}
	analyticsSvc, derived := newTestServices(analytics, &fakeBudgetRepo{}, &fakeGoalRepo{}, &fakeWalletRepo{}, &fakeBillRepo{})
	router := newAnalyticsRouter(analyticsSvc, derived)

	rec := doAnalyticsRequest(t, router, "/analytics/category-trend?category=Food&currency=PHP&months=3")
	var out struct {
		Category   string `json:"category"`
		Currency   string `json:"currency"`
		SampleSize int    `json:"sampleSize"`
		Sufficient bool   `json:"sufficient"`
		Months     []struct {
			Month       string `json:"month"`
			AmountCents int64  `json:"amountCents"`
		} `json:"months"`
	}
	decodeData(t, rec, &out)

	if out.Category != "Food" || out.Currency != "PHP" {
		t.Errorf("category/currency = %s/%s, want Food/PHP", out.Category, out.Currency)
	}
	if out.SampleSize != 3 || !out.Sufficient {
		t.Errorf("sampleSize = %d sufficient = %v, want 3 true", out.SampleSize, out.Sufficient)
	}
	if len(out.Months) != 3 {
		t.Fatalf("months = %d, want 3 (zero-filled)", len(out.Months))
	}
	// The seeded amount lands in the current month, the last of the window.
	if out.Months[2].AmountCents != 5000 {
		t.Errorf("months[2].amountCents = %d, want 5000", out.Months[2].AmountCents)
	}
	if out.Months[0].AmountCents != 0 {
		t.Errorf("months[0].amountCents = %d, want 0 (zero-filled)", out.Months[0].AmountCents)
	}
}

func TestCategoryTrendInsufficientSample(t *testing.T) {
	analyticsSvc, derived := newTestServices(&fakeAnalyticsRepo{}, &fakeBudgetRepo{}, &fakeGoalRepo{}, &fakeWalletRepo{}, &fakeBillRepo{})
	router := newAnalyticsRouter(analyticsSvc, derived)

	rec := doAnalyticsRequest(t, router, "/analytics/category-trend?category=Food&currency=PHP&months=2")
	var out struct {
		Sufficient bool `json:"sufficient"`
	}
	decodeData(t, rec, &out)
	if out.Sufficient {
		t.Error("sufficient = true, want false for a 2-month sample")
	}
}

func TestCategoryTrendRejectsNonIntegerMonths(t *testing.T) {
	analyticsSvc, derived := newTestServices(&fakeAnalyticsRepo{}, &fakeBudgetRepo{}, &fakeGoalRepo{}, &fakeWalletRepo{}, &fakeBillRepo{})
	router := newAnalyticsRouter(analyticsSvc, derived)

	rec := doAnalyticsRequest(t, router, "/analytics/category-trend?category=Food&currency=PHP&months=abc")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", rec.Code, rec.Body.String())
	}
}

// --- /budget-health ---

func TestBudgetHealthNoBudgetOmitsCurrency(t *testing.T) {
	analyticsSvc, derived := newTestServices(&fakeAnalyticsRepo{}, &fakeBudgetRepo{}, &fakeGoalRepo{}, &fakeWalletRepo{}, &fakeBillRepo{})
	router := newAnalyticsRouter(analyticsSvc, derived)

	rec := doAnalyticsRequest(t, router, "/analytics/budget-health?month=2026-01")
	var out struct {
		Month     string `json:"month"`
		Currency  string `json:"currency"`
		HasBudget bool   `json:"hasBudget"`
	}
	decodeData(t, rec, &out)

	if out.Month != "2026-01" || out.HasBudget {
		t.Errorf("month = %s hasBudget = %v, want 2026-01 false", out.Month, out.HasBudget)
	}
	if out.Currency != "" {
		t.Errorf("currency = %q, want omitted when no budget", out.Currency)
	}
}

func TestBudgetHealthWithBudget(t *testing.T) {
	analytics := &fakeAnalyticsRepo{unbudgeted: 2500}
	budget := &fakeBudgetRepo{
		budget: &domain.Budget{ID: "b-1", UserEmail: analyticsTestUser, Month: "2026-01", Currency: "PHP"},
		categories: []*domain.BudgetCategory{
			{Category: "Food", AllocatedCents: 10000},
			{Category: "Transport", AllocatedCents: 5000},
		},
		spentMap: map[string]int64{"Food": 4000, "Transport": 5000},
	}
	analyticsSvc, derived := newTestServices(analytics, budget, &fakeGoalRepo{}, &fakeWalletRepo{}, &fakeBillRepo{})
	router := newAnalyticsRouter(analyticsSvc, derived)

	rec := doAnalyticsRequest(t, router, "/analytics/budget-health?month=2026-01")
	var out struct {
		Month                string `json:"month"`
		Currency             string `json:"currency"`
		HasBudget            bool   `json:"hasBudget"`
		TotalAllocatedCents  int64  `json:"totalAllocatedCents"`
		TotalSpentCents      int64  `json:"totalSpentCents"`
		TotalRemainingCents  int64  `json:"totalRemainingCents"`
		UnbudgetedSpentCents int64  `json:"unbudgetedSpentCents"`
		Categories           []struct {
			Category       string `json:"category"`
			AllocatedCents int64  `json:"allocatedCents"`
			SpentCents     int64  `json:"spentCents"`
			RemainingCents int64  `json:"remainingCents"`
		} `json:"categories"`
	}
	decodeData(t, rec, &out)

	if !out.HasBudget || out.Currency != "PHP" {
		t.Errorf("hasBudget = %v currency = %q, want true PHP", out.HasBudget, out.Currency)
	}
	if out.TotalAllocatedCents != 15000 || out.TotalSpentCents != 9000 || out.TotalRemainingCents != 6000 {
		t.Errorf("totals = %d/%d/%d, want 15000/9000/6000", out.TotalAllocatedCents, out.TotalSpentCents, out.TotalRemainingCents)
	}
	if out.UnbudgetedSpentCents != 2500 {
		t.Errorf("unbudgetedSpentCents = %d, want 2500", out.UnbudgetedSpentCents)
	}
	if len(out.Categories) != 2 {
		t.Fatalf("categories = %d, want 2", len(out.Categories))
	}
	if out.Categories[0].RemainingCents != 6000 {
		t.Errorf("Food remaining = %d, want 6000", out.Categories[0].RemainingCents)
	}
}

// --- /goal-health ---

func TestGoalHealthRequiredMonthlyCents(t *testing.T) {
	future := time.Now().AddDate(1, 0, 0)
	goal := &fakeGoalRepo{
		goals: []*domain.SavingsGoal{
			{ID: "g-1", UserEmail: analyticsTestUser, Name: "Emergency", Currency: "PHP", TargetAmountCents: 100000, TargetDate: &future},
			{ID: "g-2", UserEmail: analyticsTestUser, Name: "No target date", Currency: "PHP", TargetAmountCents: 50000},
		},
		currents: map[string]int64{"g-1": 20000, "g-2": 0},
	}
	analyticsSvc, derived := newTestServices(&fakeAnalyticsRepo{}, &fakeBudgetRepo{}, goal, &fakeWalletRepo{}, &fakeBillRepo{})
	router := newAnalyticsRouter(analyticsSvc, derived)

	rec := doAnalyticsRequest(t, router, "/analytics/goal-health")
	var out struct {
		Goals []struct {
			ID                   string  `json:"id"`
			Name                 string  `json:"name"`
			RequiredMonthlyCents *int64  `json:"requiredMonthlyCents"`
			Status               string  `json:"status"`
		} `json:"goals"`
	}
	decodeData(t, rec, &out)

	if len(out.Goals) != 2 {
		t.Fatalf("goals = %d, want 2", len(out.Goals))
	}
	if out.Goals[0].RequiredMonthlyCents == nil {
		t.Error("goal with future target date: requiredMonthlyCents omitted, want present")
	}
	if out.Goals[1].RequiredMonthlyCents != nil {
		t.Error("goal without target date: requiredMonthlyCents present, want omitted")
	}
}

// --- /savings-rate ---

func TestSavingsRateBareArrayAndZeroIncome(t *testing.T) {
	analytics := &fakeAnalyticsRepo{
		cashFlow: []domain.CurrencyTotal{
			{Currency: "PHP", TotalCents: 3000, ExpenseCents: 7000, IncomeCents: 10000},
			{Currency: "USD", TotalCents: -500, ExpenseCents: 500, IncomeCents: 0},
		},
	}
	analyticsSvc, derived := newTestServices(analytics, &fakeBudgetRepo{}, &fakeGoalRepo{}, &fakeWalletRepo{}, &fakeBillRepo{})
	router := newAnalyticsRouter(analyticsSvc, derived)

	rec := doAnalyticsRequest(t, router, "/analytics/savings-rate?from=2026-01-01&to=2026-02-01")
	// The response data is a bare array, not an object.
	var out []struct {
		Currency     string  `json:"currency"`
		IncomeCents  int64   `json:"incomeCents"`
		ExpenseCents int64   `json:"expenseCents"`
		NetCents     int64   `json:"netCents"`
		RatePercent  float64 `json:"ratePercent"`
		ZeroIncome   bool    `json:"zeroIncome"`
	}
	decodeData(t, rec, &out)

	if len(out) != 2 {
		t.Fatalf("rates = %d, want 2", len(out))
	}
	if out[0].Currency != "PHP" || out[0].RatePercent != 30 || out[0].ZeroIncome {
		t.Errorf("PHP rate = %+v, want 30%% and zeroIncome=false", out[0])
	}
	if !out[1].ZeroIncome || out[1].RatePercent != 0 {
		t.Errorf("USD rate = %+v, want zeroIncome=true rate=0", out[1])
	}
}

// --- parseRange ---

func TestParseRangeDefaultsToCurrentMonth(t *testing.T) {
	analyticsSvc, derived := newTestServices(&fakeAnalyticsRepo{}, &fakeBudgetRepo{}, &fakeGoalRepo{}, &fakeWalletRepo{}, &fakeBillRepo{})
	router := newAnalyticsRouter(analyticsSvc, derived)

	rec := doAnalyticsRequest(t, router, "/analytics/cash-flow")
	var out struct {
		DateRange struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"dateRange"`
	}
	decodeData(t, rec, &out)

	now := time.Now().UTC()
	wantFrom := now.Format("2006-01") + "-01"
	if out.DateRange.From != wantFrom {
		t.Errorf("from = %s, want %s (current month)", out.DateRange.From, wantFrom)
	}
}

func TestParseRangeRejectsMalformedDate(t *testing.T) {
	analyticsSvc, derived := newTestServices(&fakeAnalyticsRepo{}, &fakeBudgetRepo{}, &fakeGoalRepo{}, &fakeWalletRepo{}, &fakeBillRepo{})
	router := newAnalyticsRouter(analyticsSvc, derived)

	rec := doAnalyticsRequest(t, router, "/analytics/cash-flow?from=not-a-date&to=2026-02-01")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", rec.Code, rec.Body.String())
	}
}

func TestParseRangeRejectsToBeforeFrom(t *testing.T) {
	analyticsSvc, derived := newTestServices(&fakeAnalyticsRepo{}, &fakeBudgetRepo{}, &fakeGoalRepo{}, &fakeWalletRepo{}, &fakeBillRepo{})
	router := newAnalyticsRouter(analyticsSvc, derived)

	rec := doAnalyticsRequest(t, router, "/analytics/cash-flow?from=2026-02-01&to=2026-01-01")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", rec.Code, rec.Body.String())
	}
}