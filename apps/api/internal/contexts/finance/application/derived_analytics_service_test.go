package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
	"github.com/jjspscl/my/internal/platform/timeutil"
)

func newDerivedAnalyticsService(repo *mockAnalyticsRepo, wallet *mockWalletRepo, bill *mockBillRepo) *DerivedAnalyticsService {
	loc, _ := time.LoadLocation("Asia/Manila")
	clock := timeutil.New(loc)
	analyticsSvc := NewAnalyticsService(repo, newMockBudgetRepo(), &mockGoalRepo{}).WithClock(clock)
	billSvc := NewBillService(bill).WithClock(clock)
	return NewDerivedAnalyticsService(repo, wallet, bill, analyticsSvc, billSvc).WithClock(clock)
}

// anomalySeries builds a 12-month series with an alternating 4500/5500
// baseline (so windows have nonzero MAD) and one spike at the given offset
// from the current month. spikeOffset -1 means no spike.
func anomalySeries(spikeOffset int) []domain.CategoryMonthlySpend {
	rows := make([]domain.CategoryMonthlySpend, 0, 12)
	for i := 0; i < 12; i++ {
		amount := int64(4500)
		if i%2 == 1 {
			amount = 5500
		}
		if i == spikeOffset {
			amount = 50000
		}
		rows = append(rows, domain.CategoryMonthlySpend{
			Category:    "Food",
			Currency:    "PHP",
			Month:       currentMonthOffset(i - 11),
			AmountCents: amount,
		})
	}
	return rows
}

func TestGetMonthlyAnomaliesFlagsSpike(t *testing.T) {
	repo := &mockAnalyticsRepo{categoryMonthlyAll: anomalySeries(6)}
	svc := newDerivedAnalyticsService(repo, &mockWalletRepo{}, newMockBillRepo())

	report, err := svc.GetMonthlyAnomalies(context.Background(), analyticsTestUser, "PHP", 12)
	if err != nil {
		t.Fatalf("GetMonthlyAnomalies: %v", err)
	}
	if !report.Sufficient {
		t.Error("12 months should be sufficient")
	}
	if len(report.Anomalies) != 1 {
		t.Fatalf("expected 1 anomaly, got %d", len(report.Anomalies))
	}
	a := report.Anomalies[0]
	if a.Category != "Food" || a.AmountCents != 50000 {
		t.Errorf("anomaly = %+v, want Food 50000", a)
	}
	if a.MedianCents != 5500 {
		t.Errorf("median = %d, want 5500", a.MedianCents)
	}
	if a.Ratio != 50000.0/5500.0 {
		t.Errorf("ratio = %v, want %v", a.Ratio, 50000.0/5500.0)
	}
	if !strings.Contains(a.Explanation, "₱500.00") || !strings.Contains(a.Explanation, "9.1x") {
		t.Errorf("explanation missing amount or ratio: %q", a.Explanation)
	}
}

func TestGetMonthlyAnomaliesStableSeriesNone(t *testing.T) {
	repo := &mockAnalyticsRepo{categoryMonthlyAll: anomalySeries(-1)} // no spike
	svc := newDerivedAnalyticsService(repo, &mockWalletRepo{}, newMockBillRepo())

	report, err := svc.GetMonthlyAnomalies(context.Background(), analyticsTestUser, "PHP", 12)
	if err != nil {
		t.Fatalf("GetMonthlyAnomalies: %v", err)
	}
	if len(report.Anomalies) != 0 {
		t.Fatalf("expected no anomalies, got %d", len(report.Anomalies))
	}
}

func TestGetMonthlyAnomaliesInsufficientSample(t *testing.T) {
	svc := newDerivedAnalyticsService(&mockAnalyticsRepo{}, &mockWalletRepo{}, newMockBillRepo())

	report, err := svc.GetMonthlyAnomalies(context.Background(), analyticsTestUser, "PHP", 3)
	if err != nil {
		t.Fatalf("GetMonthlyAnomalies: %v", err)
	}
	if report.Sufficient {
		t.Error("3 months must not be sufficient")
	}
	if report.Months != 3 {
		t.Errorf("months = %d, want 3", report.Months)
	}
}

func TestGetMonthlyAnomaliesFiltersCurrency(t *testing.T) {
	rows := anomalySeries(6)
	rows = append(rows, domain.CategoryMonthlySpend{
		Category: "Food", Currency: "USD", Month: currentMonthOffset(0), AmountCents: 999999,
	})
	repo := &mockAnalyticsRepo{categoryMonthlyAll: rows}
	svc := newDerivedAnalyticsService(repo, &mockWalletRepo{}, newMockBillRepo())

	report, err := svc.GetMonthlyAnomalies(context.Background(), analyticsTestUser, "PHP", 12)
	if err != nil {
		t.Fatalf("GetMonthlyAnomalies: %v", err)
	}
	if len(report.Anomalies) != 1 {
		t.Fatalf("expected only PHP anomalies, got %d", len(report.Anomalies))
	}
}

func TestGetMonthlyAnomaliesRejectsBadInput(t *testing.T) {
	svc := newDerivedAnalyticsService(&mockAnalyticsRepo{}, &mockWalletRepo{}, newMockBillRepo())

	if _, err := svc.GetMonthlyAnomalies(context.Background(), analyticsTestUser, "", 6); err == nil {
		t.Error("expected error for empty currency")
	}
	if _, err := svc.GetMonthlyAnomalies(context.Background(), analyticsTestUser, "PHP", 0); err == nil {
		t.Error("expected error for months < 1")
	}
	if _, err := svc.GetMonthlyAnomalies(context.Background(), analyticsTestUser, "PHP", 25); err == nil {
		t.Error("expected error for months > 24")
	}
}

// recurringAmounts builds expense rows for a category across the last 3 months.
func recurringAmounts(category string, amounts []int64) []domain.ExpenseAmount {
	rows := make([]domain.ExpenseAmount, 0, len(amounts))
	for i, a := range amounts {
		rows = append(rows, domain.ExpenseAmount{
			Category:    category,
			Currency:    "PHP",
			Month:       currentMonthOffset(i - 2),
			AmountCents: a,
		})
	}
	return rows
}

func TestGetRecurringChargesClassifies(t *testing.T) {
	bills := newMockBillRepo()
	bills.bills["b1"] = &domain.RecurringBill{ID: "b1", UserEmail: analyticsTestUser, Name: "Netflix", Category: "Netflix", AmountCents: 1000, Currency: "PHP"}
	bills.bills["b2"] = &domain.RecurringBill{ID: "b2", UserEmail: analyticsTestUser, Name: "Electricity", Category: "Electricity", AmountCents: 1500, Currency: "PHP"}

	repo := &mockAnalyticsRepo{
		expenseAmounts: append(append(
			recurringAmounts("Netflix", []int64{1000, 1000, 1000}),
			recurringAmounts("Electricity", []int64{1850, 1850, 1850})...),
			recurringAmounts("Gym", []int64{1200, 1200, 1200})...),
	}
	svc := newDerivedAnalyticsService(repo, &mockWalletRepo{}, bills)

	summary, err := svc.GetRecurringCharges(context.Background(), analyticsTestUser, "PHP", 6)
	if err != nil {
		t.Fatalf("GetRecurringCharges: %v", err)
	}
	if len(summary.Charges) != 3 {
		t.Fatalf("expected 3 charges, got %d", len(summary.Charges))
	}
	byCat := map[string]domain.RecurringCharge{}
	for _, c := range summary.Charges {
		byCat[c.Category] = c
	}

	if got := byCat["Netflix"]; got.Status != domain.RecurringChargeTracked || got.BillName != "Netflix" {
		t.Errorf("Netflix = %+v, want tracked against bill Netflix", got)
	}
	if got := byCat["Electricity"]; got.Status != domain.RecurringChargeAmountChanged || got.BillName != "Electricity" {
		t.Errorf("Electricity = %+v, want amount_changed against bill Electricity", got)
	}
	if got := byCat["Gym"]; got.Status != domain.RecurringChargeUntracked {
		t.Errorf("Gym = %+v, want untracked", got)
	}
}

func TestGetRecurringChargesSkipsSparseCategories(t *testing.T) {
	repo := &mockAnalyticsRepo{
		expenseAmounts: recurringAmounts("Coffee", []int64{300, 300}), // only 2 occurrences
	}
	svc := newDerivedAnalyticsService(repo, &mockWalletRepo{}, newMockBillRepo())

	summary, err := svc.GetRecurringCharges(context.Background(), analyticsTestUser, "PHP", 6)
	if err != nil {
		t.Fatalf("GetRecurringCharges: %v", err)
	}
	if len(summary.Charges) != 0 {
		t.Fatalf("expected no charges, got %d", len(summary.Charges))
	}
}

func TestGetRecurringChargesToleranceFloor(t *testing.T) {
	// A ₱20 subscription drifting by ₱2 (10%) must stay tracked: the ₱50
	// floor dominates the 10% tolerance.
	bills := newMockBillRepo()
	bills.bills["b1"] = &domain.RecurringBill{ID: "b1", UserEmail: analyticsTestUser, Name: "App", Category: "App", AmountCents: 2000, Currency: "PHP"}

	repo := &mockAnalyticsRepo{
		expenseAmounts: recurringAmounts("App", []int64{2200, 2200, 2200}),
	}
	svc := newDerivedAnalyticsService(repo, &mockWalletRepo{}, bills)

	summary, err := svc.GetRecurringCharges(context.Background(), analyticsTestUser, "PHP", 6)
	if err != nil {
		t.Fatalf("GetRecurringCharges: %v", err)
	}
	if len(summary.Charges) != 1 || summary.Charges[0].Status != domain.RecurringChargeTracked {
		t.Fatalf("charges = %+v, want App tracked (within ₱50 floor)", summary.Charges)
	}
}

func TestGetBillReconciliation(t *testing.T) {
	start, _ := time.Parse("2006-01-02", "2026-01-01")
	repo := &mockAnalyticsRepo{
		billReconciliation: []domain.BillReconciliationRow{
			{BillID: "b1", Name: "Rent", Category: "Housing", Currency: "PHP", AmountCents: 10000, Frequency: domain.FrequencyMonthly, DayOfMonth: 1, StartDate: start, PaidCents: 10000, PaidCount: 1},
			{BillID: "b2", Name: "Electricity", Category: "Utilities", Currency: "PHP", AmountCents: 1000, Frequency: domain.FrequencyMonthly, DayOfMonth: 5, StartDate: start, PaidCents: 1200, PaidCount: 1, PaidWithoutTransactionCount: 1},
		},
	}
	svc := newDerivedAnalyticsService(repo, &mockWalletRepo{}, newMockBillRepo())

	recon, err := svc.GetBillReconciliation(context.Background(), analyticsTestUser, "2026-07")
	if err != nil {
		t.Fatalf("GetBillReconciliation: %v", err)
	}
	if len(recon.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(recon.Items))
	}
	byName := map[string]domain.BillReconciliationItem{}
	for _, it := range recon.Items {
		byName[it.Name] = it
	}
	if got := byName["Rent"]; got.ExpectedCents != 10000 || got.VarianceCents != 0 {
		t.Errorf("Rent = %+v, want expected 10000 variance 0", got)
	}
	if got := byName["Electricity"]; got.ExpectedCents != 1000 || got.VarianceCents != 200 || got.PaidWithoutTransactionCount != 1 {
		t.Errorf("Electricity = %+v, want expected 1000 variance 200 paidWithoutTx 1", got)
	}
	if !strings.Contains(byName["Electricity"].Explanation, "no linked transaction") {
		t.Errorf("Electricity explanation should mention missing transaction: %q", byName["Electricity"].Explanation)
	}
}

func TestGetBillReconciliationWeeklyOccurrences(t *testing.T) {
	start, _ := time.Parse("2006-01-02", "2026-07-01")
	repo := &mockAnalyticsRepo{
		billReconciliation: []domain.BillReconciliationRow{
			{BillID: "b1", Name: "Gym", Category: "Health", Currency: "PHP", AmountCents: 500, Frequency: domain.FrequencyWeekly, StartDate: start, PaidCents: 2000, PaidCount: 4},
		},
	}
	svc := newDerivedAnalyticsService(repo, &mockWalletRepo{}, newMockBillRepo())

	recon, err := svc.GetBillReconciliation(context.Background(), analyticsTestUser, "2026-07")
	if err != nil {
		t.Fatalf("GetBillReconciliation: %v", err)
	}
	if len(recon.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(recon.Items))
	}
	// July 2026 has 31 days: weekly occurrences on the 1st, 8th, 15th, 22nd,
	// and 29th = 5 × 500 = 2500 expected.
	if recon.Items[0].ExpectedCents != 2500 || recon.Items[0].VarianceCents != -500 {
		t.Errorf("item = %+v, want expected 2500 variance -500", recon.Items[0])
	}
}

// essentialSeries builds 12 months of essential spend for one currency.
func essentialSeries(amount int64) []domain.MonthlyEssentialSpend {
	rows := make([]domain.MonthlyEssentialSpend, 0, 12)
	for i := 0; i < 12; i++ {
		rows = append(rows, domain.MonthlyEssentialSpend{
			Currency:    "PHP",
			Month:       currentMonthOffset(i - 11),
			AmountCents: amount,
		})
	}
	return rows
}

func TestGetEmergencyFund(t *testing.T) {
	repo := &mockAnalyticsRepo{essentialMonthly: essentialSeries(30000)}
	wallet := &mockWalletRepo{balances: []*domain.WalletBalance{
		{Wallet: domain.Wallet{Currency: "PHP"}, BalanceCents: 60000},
	}}
	svc := newDerivedAnalyticsService(repo, wallet, newMockBillRepo())

	status, err := svc.GetEmergencyFund(context.Background(), analyticsTestUser, "PHP", 0)
	if err != nil {
		t.Fatalf("GetEmergencyFund: %v", err)
	}
	if status.MonthlyEssentialCents != 30000 {
		t.Errorf("monthly essential = %d, want 30000", status.MonthlyEssentialCents)
	}
	if status.LiquidBalanceCents != 60000 {
		t.Errorf("liquid = %d, want 60000", status.LiquidBalanceCents)
	}
	if status.MonthsOfRunway != 2.0 {
		t.Errorf("runway = %v, want 2.0", status.MonthsOfRunway)
	}
	if status.TargetRangeMonths != [2]int{3, 6} {
		t.Errorf("target = %v, want [3 6]", status.TargetRangeMonths)
	}
	if status.ShortfallToMinCents != 30000 || status.ShortfallToMaxCents != 120000 {
		t.Errorf("shortfall = %d/%d, want 30000/120000", status.ShortfallToMinCents, status.ShortfallToMaxCents)
	}
	if len(status.Assumptions) == 0 {
		t.Error("expected named assumptions")
	}
}

func TestGetEmergencyFundTargetOverride(t *testing.T) {
	repo := &mockAnalyticsRepo{essentialMonthly: essentialSeries(30000)}
	wallet := &mockWalletRepo{balances: []*domain.WalletBalance{
		{Wallet: domain.Wallet{Currency: "PHP"}, BalanceCents: 60000},
	}}
	svc := newDerivedAnalyticsService(repo, wallet, newMockBillRepo())

	status, err := svc.GetEmergencyFund(context.Background(), analyticsTestUser, "PHP", 12)
	if err != nil {
		t.Fatalf("GetEmergencyFund: %v", err)
	}
	if status.TargetRangeMonths != [2]int{12, 12} {
		t.Errorf("target = %v, want [12 12]", status.TargetRangeMonths)
	}
	if status.ShortfallToMinCents != 300000 {
		t.Errorf("shortfall = %d, want 300000", status.ShortfallToMinCents)
	}
}

func TestGetEmergencyFundInsufficientClassification(t *testing.T) {
	repo := &mockAnalyticsRepo{
		essentialMonthly: essentialSeries(30000),
		unclassified: []domain.UnclassifiedSpending{
			{Currency: "PHP", TotalCents: 100000, UnclassifiedCents: 50000}, // 50% unclassified
		},
		topUnclassified: []domain.CategorySpend{{Category: "Misc", AmountCents: 50000}},
	}
	svc := newDerivedAnalyticsService(repo, &mockWalletRepo{}, newMockBillRepo())

	_, err := svc.GetEmergencyFund(context.Background(), analyticsTestUser, "PHP", 0)
	var insufficient *domain.ErrInsufficientClassification
	if !errors.As(err, &insufficient) {
		t.Fatalf("expected ErrInsufficientClassification, got %v", err)
	}
	if len(insufficient.TopUnclassified) != 1 || insufficient.TopUnclassified[0] != "Misc" {
		t.Errorf("top unclassified = %v, want [Misc]", insufficient.TopUnclassified)
	}
}

func TestGetEmergencyFundRejectsBadInput(t *testing.T) {
	svc := newDerivedAnalyticsService(&mockAnalyticsRepo{}, &mockWalletRepo{}, newMockBillRepo())

	if _, err := svc.GetEmergencyFund(context.Background(), analyticsTestUser, "", 0); err == nil {
		t.Error("expected error for empty currency")
	}
	if _, err := svc.GetEmergencyFund(context.Background(), analyticsTestUser, "PHP", 13); err == nil {
		t.Error("expected error for targetMonths > 12")
	}
}

func TestGetAffordability(t *testing.T) {
	repo := &mockAnalyticsRepo{essentialMonthly: essentialSeries(30000)}
	wallet := &mockWalletRepo{balances: []*domain.WalletBalance{
		{Wallet: domain.Wallet{Currency: "PHP"}, BalanceCents: 60000},
	}}
	bills := newMockBillRepo()
	start, _ := time.Parse("2006-01-02", "2026-01-01")
	bills.bills["b1"] = &domain.RecurringBill{
		ID: "b1", UserEmail: analyticsTestUser, Name: "Rent", Category: "Housing",
		AmountCents: 30000, Currency: "PHP", Frequency: domain.FrequencyMonthly, DayOfMonth: 1, StartDate: start,
	}
	svc := newDerivedAnalyticsService(repo, wallet, bills)

	model, err := svc.GetAffordability(context.Background(), analyticsTestUser, "PHP", 20000)
	if err != nil {
		t.Fatalf("GetAffordability: %v", err)
	}
	if model.MonthlyEssentialCents != 30000 {
		t.Errorf("monthly essential = %d, want 30000", model.MonthlyEssentialCents)
	}
	if model.UpcomingBillsCents != 30000 {
		t.Errorf("upcoming bills = %d, want 30000", model.UpcomingBillsCents)
	}
	if model.MonthlyObligationCents != 60000 {
		t.Errorf("obligation = %d, want 60000", model.MonthlyObligationCents)
	}
	if model.RunwayMonthsBefore != 1.0 {
		t.Errorf("runway before = %v, want 1.0", model.RunwayMonthsBefore)
	}
	if model.RunwayMonthsAfter != 40000.0/60000.0 {
		t.Errorf("runway after = %v, want %v", model.RunwayMonthsAfter, 40000.0/60000.0)
	}
}

func TestGetAffordabilityRejectsBadInput(t *testing.T) {
	svc := newDerivedAnalyticsService(&mockAnalyticsRepo{}, &mockWalletRepo{}, newMockBillRepo())

	if _, err := svc.GetAffordability(context.Background(), analyticsTestUser, "", 1000); err == nil {
		t.Error("expected error for empty currency")
	}
	if _, err := svc.GetAffordability(context.Background(), analyticsTestUser, "PHP", 0); err == nil {
		t.Error("expected error for non-positive amount")
	}
}

func TestGetMonthlyDigest(t *testing.T) {
	repo := &mockAnalyticsRepo{
		cashFlow: []domain.CurrencyTotal{
			{Currency: "PHP", IncomeCents: 100000, ExpenseCents: 40000, TotalCents: 60000},
		},
		monthlyFlow: []domain.MonthlyCashFlow{
			{Month: currentMonthOffset(0), Currency: "PHP", IncomeCents: 100000, ExpenseCents: 40000, NetCents: 60000},
		},
		classification: []domain.ClassificationSpend{
			{Currency: "PHP", Classification: domain.ClassificationNeeds, AmountCents: 30000},
			{Currency: "PHP", Classification: domain.ClassificationWants, AmountCents: 10000},
		},
		unclassified: []domain.UnclassifiedSpending{
			{Currency: "PHP", TotalCents: 40000, UnclassifiedCents: 0},
		},
		essentialMonthly:   essentialSeries(30000),
		expenseAmounts:     recurringAmounts("Netflix", []int64{1000, 1000, 1000}),
		categoryMonthlyAll: anomalySeries(-1),
	}
	wallet := &mockWalletRepo{balances: []*domain.WalletBalance{
		{Wallet: domain.Wallet{Currency: "PHP"}, BalanceCents: 60000},
	}}
	svc := newDerivedAnalyticsService(repo, wallet, newMockBillRepo())

	digest, err := svc.GetMonthlyDigest(context.Background(), analyticsTestUser, currentMonthOffset(0))
	if err != nil {
		t.Fatalf("GetMonthlyDigest: %v", err)
	}
	if !digest.CashFlow.Present || !strings.Contains(digest.CashFlow.Summary, "income ₱1,000.00") {
		t.Errorf("cash flow section = %+v", digest.CashFlow)
	}
	if !digest.Spending.Present || !strings.Contains(digest.Spending.Summary, "needs") {
		t.Errorf("spending section = %+v", digest.Spending)
	}
	if !digest.SavingsRate.Present || !strings.Contains(digest.SavingsRate.Summary, "60.0%") {
		t.Errorf("savings section = %+v", digest.SavingsRate)
	}
	if !digest.Recurring.Present || len(digest.Recurring.Charges) != 1 {
		t.Errorf("recurring section = %+v", digest.Recurring)
	}
	if !digest.Anomalies.Present {
		t.Errorf("anomalies section = %+v", digest.Anomalies)
	}
	if !digest.Emergency.Present || digest.Emergency.Status.MonthsOfRunway != 2.0 {
		t.Errorf("emergency section = %+v", digest.Emergency)
	}
	if len(digest.Omitted) != 0 {
		t.Errorf("omitted = %v, want none", digest.Omitted)
	}
}

func TestGetMonthlyDigestOmitsSpendingOnInsufficientClassification(t *testing.T) {
	repo := &mockAnalyticsRepo{
		cashFlow: []domain.CurrencyTotal{
			{Currency: "PHP", IncomeCents: 100000, ExpenseCents: 40000, TotalCents: 60000},
		},
		monthlyFlow: []domain.MonthlyCashFlow{
			{Month: currentMonthOffset(0), Currency: "PHP", IncomeCents: 100000, ExpenseCents: 40000, NetCents: 60000},
		},
		classification: []domain.ClassificationSpend{
			{Currency: "PHP", Classification: domain.ClassificationNeeds, AmountCents: 20000},
		},
		unclassified: []domain.UnclassifiedSpending{
			{Currency: "PHP", TotalCents: 40000, UnclassifiedCents: 20000}, // 50% unclassified
		},
		topUnclassified:  []domain.CategorySpend{{Category: "Misc", AmountCents: 20000}},
		essentialMonthly: essentialSeries(30000),
	}
	wallet := &mockWalletRepo{balances: []*domain.WalletBalance{
		{Wallet: domain.Wallet{Currency: "PHP"}, BalanceCents: 60000},
	}}
	svc := newDerivedAnalyticsService(repo, wallet, newMockBillRepo())

	digest, err := svc.GetMonthlyDigest(context.Background(), analyticsTestUser, currentMonthOffset(0))
	if err != nil {
		t.Fatalf("GetMonthlyDigest: %v", err)
	}
	if digest.Spending.Present {
		t.Error("spending section must be omitted when classification is insufficient")
	}
	found := false
	for _, o := range digest.Omitted {
		if strings.Contains(o, "spending breakdown") {
			found = true
		}
	}
	if !found {
		t.Errorf("omitted = %v, want spending breakdown reason", digest.Omitted)
	}
}
