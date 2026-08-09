package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
)

// monthsInWindow returns the last n YYYY-MM months ending with the current
// month, matching how the services compute their windows from the real clock.
func monthsInWindow(n int) []string {
	now := time.Now().UTC()
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = now.AddDate(0, i-(n-1), 0).Format("2006-01")
	}
	return out
}

// essentialSpendRows seeds every month of the essential-spend window with the
// same amount. The median is computed over the zero-filled window, so a single
// seeded month would be diluted by zeros.
func essentialSpendRows(currency string, amountCents int64) []domain.MonthlyEssentialSpend {
	months := monthsInWindow(domain.EmergencyFundWindowMonths)
	rows := make([]domain.MonthlyEssentialSpend, 0, len(months))
	for _, m := range months {
		rows = append(rows, domain.MonthlyEssentialSpend{Currency: currency, Month: m, AmountCents: amountCents})
	}
	return rows
}

// --- /anomalies ---

func TestAnomaliesFlagsOutlier(t *testing.T) {
	months := monthsInWindow(6)
	// A spread baseline with a single spike in the current month. The baseline
	// must have non-zero MAD or the Hampel filter skips the point.
	rows := make([]domain.CategoryMonthlySpend, 0, 6)
	for i, m := range months {
		amount := int64(100)
		if i%2 == 1 {
			amount = 200
		}
		if i == 5 {
			amount = 10000
		}
		rows = append(rows, domain.CategoryMonthlySpend{Category: "Food", Currency: "PHP", Month: m, AmountCents: amount})
	}
	analytics := &fakeAnalyticsRepo{categoryMonthlyAll: rows}
	analyticsSvc, derived := newTestServices(analytics, &fakeBudgetRepo{}, &fakeGoalRepo{}, &fakeWalletRepo{}, &fakeBillRepo{})
	router := newAnalyticsRouter(analyticsSvc, derived)

	rec := doAnalyticsRequest(t, router, "/analytics/anomalies?currency=PHP&months=6")
	var out struct {
		Currency   string `json:"currency"`
		Months     int    `json:"months"`
		Sufficient bool   `json:"sufficient"`
		Anomalies  []struct {
			Category    string  `json:"category"`
			Currency    string  `json:"currency"`
			Month       string  `json:"month"`
			AmountCents int64   `json:"amountCents"`
			MedianCents int64   `json:"medianCents"`
			Ratio       float64 `json:"ratio"`
			Explanation string  `json:"explanation"`
		} `json:"anomalies"`
	}
	decodeData(t, rec, &out)

	if out.Currency != "PHP" || out.Months != 6 || !out.Sufficient {
		t.Errorf("currency = %s months = %d sufficient = %v, want PHP 6 true", out.Currency, out.Months, out.Sufficient)
	}
	if len(out.Anomalies) != 1 {
		t.Fatalf("anomalies = %d, want 1; body: %s", len(out.Anomalies), rec.Body.String())
	}
	a := out.Anomalies[0]
	if a.Category != "Food" || a.AmountCents != 10000 {
		t.Errorf("anomaly = %+v, want Food 10000", a)
	}
	if a.Explanation == "" {
		t.Error("anomaly explanation empty, want human-readable text")
	}
}

func TestAnomaliesInsufficientSample(t *testing.T) {
	analyticsSvc, derived := newTestServices(&fakeAnalyticsRepo{}, &fakeBudgetRepo{}, &fakeGoalRepo{}, &fakeWalletRepo{}, &fakeBillRepo{})
	router := newAnalyticsRouter(analyticsSvc, derived)

	rec := doAnalyticsRequest(t, router, "/analytics/anomalies?currency=PHP&months=2")
	var out struct {
		Sufficient bool `json:"sufficient"`
	}
	decodeData(t, rec, &out)
	if out.Sufficient {
		t.Error("sufficient = true, want false for a 2-month window")
	}
}

func TestAnomaliesRejectsNonIntegerMonths(t *testing.T) {
	analyticsSvc, derived := newTestServices(&fakeAnalyticsRepo{}, &fakeBudgetRepo{}, &fakeGoalRepo{}, &fakeWalletRepo{}, &fakeBillRepo{})
	router := newAnalyticsRouter(analyticsSvc, derived)

	rec := doAnalyticsRequest(t, router, "/analytics/anomalies?currency=PHP&months=abc")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", rec.Code, rec.Body.String())
	}
}

// --- /recurring-charges ---

func TestRecurringChargesTrackedAndUntracked(t *testing.T) {
	months := monthsInWindow(3)
	amounts := make([]domain.ExpenseAmount, 0, 3)
	for _, m := range months {
		amounts = append(amounts, domain.ExpenseAmount{Category: "Netflix", Currency: "PHP", Month: m, AmountCents: 1500})
	}
	analytics := &fakeAnalyticsRepo{expenseAmounts: amounts}
	bill := &fakeBillRepo{
		bills: []*domain.RecurringBill{
			{ID: "b-1", UserEmail: analyticsTestUser, Name: "Netflix", Category: "Netflix", AmountCents: 1500, Currency: "PHP"},
		},
	}
	analyticsSvc, derived := newTestServices(analytics, &fakeBudgetRepo{}, &fakeGoalRepo{}, &fakeWalletRepo{}, bill)
	router := newAnalyticsRouter(analyticsSvc, derived)

	rec := doAnalyticsRequest(t, router, "/analytics/recurring-charges?currency=PHP&months=3")
	var out struct {
		Currency string `json:"currency"`
		Months   int    `json:"months"`
		Charges  []struct {
			Category        string `json:"category"`
			Occurrences     int    `json:"occurrences"`
			DistinctMonths  int    `json:"distinctMonths"`
			MedianCents     int64  `json:"medianCents"`
			Status          string `json:"status"`
			BillName        string `json:"billName"`
			BillAmountCents int64  `json:"billAmountCents"`
			Explanation     string `json:"explanation"`
		} `json:"charges"`
	}
	decodeData(t, rec, &out)

	if len(out.Charges) != 1 {
		t.Fatalf("charges = %d, want 1; body: %s", len(out.Charges), rec.Body.String())
	}
	c := out.Charges[0]
	if c.Status != "tracked" || c.BillName != "Netflix" || c.BillAmountCents != 1500 {
		t.Errorf("charge = %+v, want tracked Netflix 1500", c)
	}
	if c.Occurrences != 3 || c.DistinctMonths != 3 {
		t.Errorf("occurrences = %d distinctMonths = %d, want 3/3", c.Occurrences, c.DistinctMonths)
	}
}

func TestRecurringChargesUntrackedOmitsBillFields(t *testing.T) {
	months := monthsInWindow(3)
	amounts := make([]domain.ExpenseAmount, 0, 3)
	for _, m := range months {
		amounts = append(amounts, domain.ExpenseAmount{Category: "Gym", Currency: "PHP", Month: m, AmountCents: 2000})
	}
	analytics := &fakeAnalyticsRepo{expenseAmounts: amounts}
	analyticsSvc, derived := newTestServices(analytics, &fakeBudgetRepo{}, &fakeGoalRepo{}, &fakeWalletRepo{}, &fakeBillRepo{})
	router := newAnalyticsRouter(analyticsSvc, derived)

	rec := doAnalyticsRequest(t, router, "/analytics/recurring-charges?currency=PHP&months=3")
	var out struct {
		Charges []struct {
			Status          string `json:"status"`
			BillName        string `json:"billName"`
			BillAmountCents int64  `json:"billAmountCents"`
		} `json:"charges"`
	}
	decodeData(t, rec, &out)

	if len(out.Charges) != 1 || out.Charges[0].Status != "untracked" {
		t.Fatalf("charges = %+v, want one untracked", out.Charges)
	}
	// omitempty: billName and billAmountCents must be absent from the wire.
	if strings.Contains(rec.Body.String(), `"billName"`) || strings.Contains(rec.Body.String(), `"billAmountCents"`) {
		t.Errorf("untracked charge leaks bill fields: %s", rec.Body.String())
	}
}

// --- /bill-reconciliation ---

func TestBillReconciliationMapsVariance(t *testing.T) {
	analytics := &fakeAnalyticsRepo{
		billReconciliation: []domain.BillReconciliationRow{
			{
				BillID: "b-1", Name: "Rent", Category: "Housing", Currency: "PHP",
				AmountCents: 15000, Frequency: domain.FrequencyMonthly,
				StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				PaidCents: 15000, PaidCount: 1,
			},
			{
				BillID: "b-2", Name: "Internet", Category: "Utilities", Currency: "PHP",
				AmountCents: 2000, Frequency: domain.FrequencyMonthly,
				StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				PaidCents: 1500, PaidCount: 1, PaidWithoutTransactionCount: 1,
			},
		},
	}
	analyticsSvc, derived := newTestServices(analytics, &fakeBudgetRepo{}, &fakeGoalRepo{}, &fakeWalletRepo{}, &fakeBillRepo{})
	router := newAnalyticsRouter(analyticsSvc, derived)

	rec := doAnalyticsRequest(t, router, "/analytics/bill-reconciliation?month=2026-01")
	var out struct {
		Month string `json:"month"`
		Items []struct {
			BillID                      string `json:"billId"`
			Name                        string `json:"name"`
			ExpectedCents               int64  `json:"expectedCents"`
			PaidCents                   int64  `json:"paidCents"`
			VarianceCents               int64  `json:"varianceCents"`
			PaidCount                   int    `json:"paidCount"`
			PaidWithoutTransactionCount int    `json:"paidWithoutTransactionCount"`
			Explanation                 string `json:"explanation"`
		} `json:"items"`
	}
	decodeData(t, rec, &out)

	if out.Month != "2026-01" || len(out.Items) != 2 {
		t.Fatalf("month = %s items = %d, want 2026-01 2", out.Month, len(out.Items))
	}
	rent := out.Items[0]
	if rent.ExpectedCents != 15000 || rent.PaidCents != 15000 || rent.VarianceCents != 0 {
		t.Errorf("rent = %+v, want expected=paid=15000 variance=0", rent)
	}
	net := out.Items[1]
	if net.ExpectedCents != 2000 || net.PaidCents != 1500 || net.VarianceCents != -500 {
		t.Errorf("internet = %+v, want expected=2000 paid=1500 variance=-500", net)
	}
	if net.PaidWithoutTransactionCount != 1 {
		t.Errorf("paidWithoutTransactionCount = %d, want 1", net.PaidWithoutTransactionCount)
	}
}

// --- /emergency-fund ---

func TestEmergencyFundTargetRangeMonthsArray(t *testing.T) {
	analytics := &fakeAnalyticsRepo{
		unclassified: []domain.UnclassifiedSpending{
			{Currency: "PHP", UnclassifiedCents: 0, TotalCents: 10000},
		},
		essentialMonthly: essentialSpendRows("PHP", 10000),
	}
	wallet := &fakeWalletRepo{
		balances: []*domain.WalletBalance{
			{Wallet: domain.Wallet{Currency: "PHP"}, BalanceCents: 50000},
		},
	}
	analyticsSvc, derived := newTestServices(analytics, &fakeBudgetRepo{}, &fakeGoalRepo{}, wallet, &fakeBillRepo{})
	router := newAnalyticsRouter(analyticsSvc, derived)

	rec := doAnalyticsRequest(t, router, "/analytics/emergency-fund?currency=PHP&targetMonths=4")
	var out struct {
		Currency              string  `json:"currency"`
		LiquidBalanceCents    int64   `json:"liquidBalanceCents"`
		MonthlyEssentialCents int64   `json:"monthlyEssentialCents"`
		MonthsOfRunway        float64 `json:"monthsOfRunway"`
		TargetRangeMonths     [2]int  `json:"targetRangeMonths"`
		ShortfallToMinCents   int64   `json:"shortfallToMinCents"`
		ShortfallToMaxCents   int64   `json:"shortfallToMaxCents"`
	}
	decodeData(t, rec, &out)

	if out.TargetRangeMonths != [2]int{4, 4} {
		t.Errorf("targetRangeMonths = %v, want [4 4] from targetMonths=4", out.TargetRangeMonths)
	}
	if out.LiquidBalanceCents != 50000 || out.MonthlyEssentialCents != 10000 {
		t.Errorf("liquid = %d essential = %d, want 50000/10000", out.LiquidBalanceCents, out.MonthlyEssentialCents)
	}
	if out.MonthsOfRunway != 5 {
		t.Errorf("monthsOfRunway = %v, want 5", out.MonthsOfRunway)
	}
}

func TestEmergencyFundRejectsBadTargetMonths(t *testing.T) {
	analyticsSvc, derived := newTestServices(&fakeAnalyticsRepo{}, &fakeBudgetRepo{}, &fakeGoalRepo{}, &fakeWalletRepo{}, &fakeBillRepo{})
	router := newAnalyticsRouter(analyticsSvc, derived)

	rec := doAnalyticsRequest(t, router, "/analytics/emergency-fund?currency=PHP&targetMonths=abc")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", rec.Code, rec.Body.String())
	}
}

// --- /affordability ---

func TestAffordabilityRequiresAmountCents(t *testing.T) {
	analyticsSvc, derived := newTestServices(&fakeAnalyticsRepo{}, &fakeBudgetRepo{}, &fakeGoalRepo{}, &fakeWalletRepo{}, &fakeBillRepo{})
	router := newAnalyticsRouter(analyticsSvc, derived)

	rec := doAnalyticsRequest(t, router, "/analytics/affordability?currency=PHP")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 when amountCents missing; body: %s", rec.Code, rec.Body.String())
	}
}

func TestAffordabilityModel(t *testing.T) {
	analytics := &fakeAnalyticsRepo{
		unclassified: []domain.UnclassifiedSpending{
			{Currency: "PHP", UnclassifiedCents: 0, TotalCents: 10000},
		},
		essentialMonthly: essentialSpendRows("PHP", 10000),
	}
	wallet := &fakeWalletRepo{
		balances: []*domain.WalletBalance{
			{Wallet: domain.Wallet{Currency: "PHP"}, BalanceCents: 50000},
		},
	}
	analyticsSvc, derived := newTestServices(analytics, &fakeBudgetRepo{}, &fakeGoalRepo{}, wallet, &fakeBillRepo{})
	router := newAnalyticsRouter(analyticsSvc, derived)

	rec := doAnalyticsRequest(t, router, "/analytics/affordability?currency=PHP&amountCents=10000")
	var out struct {
		Currency               string  `json:"currency"`
		AmountCents            int64   `json:"amountCents"`
		LiquidBalanceCents     int64   `json:"liquidBalanceCents"`
		MonthlyEssentialCents  int64   `json:"monthlyEssentialCents"`
		UpcomingBillsCents     int64   `json:"upcomingBillsCents"`
		MonthlyObligationCents int64   `json:"monthlyObligationCents"`
		RunwayMonthsBefore     float64 `json:"runwayMonthsBefore"`
		RunwayMonthsAfter      float64 `json:"runwayMonthsAfter"`
	}
	decodeData(t, rec, &out)

	if out.AmountCents != 10000 || out.LiquidBalanceCents != 50000 || out.MonthlyEssentialCents != 10000 {
		t.Errorf("model = %+v, want amount=10000 liquid=50000 essential=10000", out)
	}
	if out.RunwayMonthsBefore != 5 || out.RunwayMonthsAfter != 4 {
		t.Errorf("runway before/after = %v/%v, want 5/4", out.RunwayMonthsBefore, out.RunwayMonthsAfter)
	}
}

// --- /digest ---

func TestDigestAllSectionsPresent(t *testing.T) {
	analytics := &fakeAnalyticsRepo{
		cashFlow: []domain.CurrencyTotal{
			{Currency: "PHP", TotalCents: 3000, ExpenseCents: 7000, IncomeCents: 10000},
		},
		monthlyFlow: []domain.MonthlyCashFlow{
			{Month: "2026-01", Currency: "PHP", IncomeCents: 10000, ExpenseCents: 7000, NetCents: 3000},
		},
		classification: []domain.ClassificationSpend{
			{Currency: "PHP", Classification: domain.ClassificationNeeds, AmountCents: 7000},
		},
		unclassified: []domain.UnclassifiedSpending{
			{Currency: "PHP", UnclassifiedCents: 0, TotalCents: 7000},
		},
		essentialMonthly: []domain.MonthlyEssentialSpend{
			{Currency: "PHP", Month: monthsInWindow(1)[0], AmountCents: 7000},
		},
	}
	wallet := &fakeWalletRepo{
		balances: []*domain.WalletBalance{
			{Wallet: domain.Wallet{Currency: "PHP"}, BalanceCents: 50000},
		},
	}
	analyticsSvc, derived := newTestServices(analytics, &fakeBudgetRepo{}, &fakeGoalRepo{}, wallet, &fakeBillRepo{})
	router := newAnalyticsRouter(analyticsSvc, derived)

	rec := doAnalyticsRequest(t, router, "/analytics/digest?month=2026-01")
	var out struct {
		Month       string `json:"month"`
		CashFlow    struct {
			Present bool   `json:"present"`
			Summary string `json:"summary"`
		} `json:"cashFlow"`
		Spending struct {
			Present bool   `json:"present"`
			Summary string `json:"summary"`
		} `json:"spending"`
		SavingsRate struct {
			Present bool   `json:"present"`
			Summary string `json:"summary"`
		} `json:"savingsRate"`
		Recurring struct {
			Present bool   `json:"present"`
			Summary string `json:"summary"`
		} `json:"recurring"`
		Anomalies struct {
			Present bool   `json:"present"`
			Summary string `json:"summary"`
		} `json:"anomalies"`
		Emergency struct {
			Present bool                   `json:"present"`
			Summary string                 `json:"summary"`
			Status  *json.RawMessage       `json:"status"`
		} `json:"emergency"`
		Omitted []string `json:"omitted"`
	}
	decodeData(t, rec, &out)

	if out.Month != "2026-01" {
		t.Errorf("month = %s, want 2026-01", out.Month)
	}
	for name, present := range map[string]bool{
		"cashFlow":    out.CashFlow.Present,
		"spending":    out.Spending.Present,
		"savingsRate": out.SavingsRate.Present,
		"recurring":   out.Recurring.Present,
		"anomalies":   out.Anomalies.Present,
		"emergency":   out.Emergency.Present,
	} {
		if !present {
			t.Errorf("%s.present = false, want true", name)
		}
	}
	if out.Emergency.Status == nil {
		t.Error("emergency.status omitted, want present when emergency computed")
	}
	if len(out.Omitted) != 0 {
		t.Errorf("omitted = %v, want empty", out.Omitted)
	}
}

func TestDigestOmitsEmergencyOnInsufficientClassification(t *testing.T) {
	analytics := &fakeAnalyticsRepo{
		cashFlow: []domain.CurrencyTotal{
			{Currency: "PHP", TotalCents: 3000, ExpenseCents: 7000, IncomeCents: 10000},
		},
		monthlyFlow: []domain.MonthlyCashFlow{
			{Month: "2026-01", Currency: "PHP", IncomeCents: 10000, ExpenseCents: 7000, NetCents: 3000},
		},
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

	rec := doAnalyticsRequest(t, router, "/analytics/digest?month=2026-01")
	var out struct {
		Spending struct {
			Present bool `json:"present"`
		} `json:"spending"`
		Emergency struct {
			Present bool             `json:"present"`
			Status  *json.RawMessage `json:"status"`
		} `json:"emergency"`
		Omitted []string `json:"omitted"`
	}
	decodeData(t, rec, &out)

	if out.Spending.Present {
		t.Error("spending.present = true, want false when classification insufficient")
	}
	if out.Emergency.Present {
		t.Error("emergency.present = true, want false when classification insufficient")
	}
	if out.Emergency.Status != nil {
		t.Error("emergency.status present, want omitted")
	}
	if len(out.Omitted) == 0 {
		t.Error("omitted empty, want reasons for spending and emergency")
	}
}