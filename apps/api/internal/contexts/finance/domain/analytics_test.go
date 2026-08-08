package domain

import (
	"strings"
	"testing"
)

func TestComputeSavingsRate(t *testing.T) {
	tests := []struct {
		name        string
		income      int64
		expense     int64
		wantRate    float64
		wantZero    bool
	}{
		{name: "positive savings", income: 10000, expense: 6000, wantRate: 40},
		{name: "break even", income: 10000, expense: 10000, wantRate: 0},
		{name: "overspend", income: 10000, expense: 12000, wantRate: -20},
		{name: "no income", income: 0, expense: 5000, wantZero: true},
		{name: "negative income impossible but guarded", income: -1, expense: 0, wantZero: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rate, zero := ComputeSavingsRate(tt.income, tt.expense)
			if zero != tt.wantZero {
				t.Errorf("zeroIncome = %v, want %v", zero, tt.wantZero)
			}
			if !zero && rate != tt.wantRate {
				t.Errorf("rate = %v, want %v", rate, tt.wantRate)
			}
		})
	}
}

func TestErrInsufficientClassificationError(t *testing.T) {
	err := &ErrInsufficientClassification{
		UnclassifiedSharePct: 42.5,
		TopUnclassified:      []string{"Food", "Shopping"},
	}
	msg := err.Error()
	if !strings.Contains(msg, "42.5%") {
		t.Errorf("error should carry the share, got: %s", msg)
	}
	if !strings.Contains(msg, "Food") || !strings.Contains(msg, "Shopping") {
		t.Errorf("error should list top unclassified categories: %s", msg)
	}
}

func TestMaxUnclassifiedShareConstant(t *testing.T) {
	if MaxUnclassifiedShare != 0.20 {
		t.Errorf("MaxUnclassifiedShare = %v, want 0.20", MaxUnclassifiedShare)
	}
	if MinTrendMonths != 3 {
		t.Errorf("MinTrendMonths = %d, want 3", MinTrendMonths)
	}
}