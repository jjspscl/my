package application

import (
	"context"
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
