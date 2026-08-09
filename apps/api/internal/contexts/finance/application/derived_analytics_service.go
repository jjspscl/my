package application

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
	"github.com/jjspscl/my/internal/platform/timeutil"
)

// DerivedAnalyticsService composes the analytics core into derived views:
// spending anomalies, recurring-charge summaries, bill reconciliation,
// emergency-fund status, affordability, and the monthly digest. It is separate
// from AnalyticsService because these views combine multiple repositories and
// services (wallet balances, bills, the base analytics) and would otherwise
// bloat the core service.
//
// Every response carries named assumptions and a human-readable explanation;
// nothing here returns an opaque score or a bare yes/no.
type DerivedAnalyticsService struct {
	analyticsRepo domain.AnalyticsRepository
	walletRepo    domain.WalletRepository
	billRepo      domain.BillRepository
	analyticsSvc  *AnalyticsService
	billSvc       *BillService
	clock         *timeutil.Clock
}

func NewDerivedAnalyticsService(
	analyticsRepo domain.AnalyticsRepository,
	walletRepo domain.WalletRepository,
	billRepo domain.BillRepository,
	analyticsSvc *AnalyticsService,
	billSvc *BillService,
) *DerivedAnalyticsService {
	return &DerivedAnalyticsService{
		analyticsRepo: analyticsRepo,
		walletRepo:    walletRepo,
		billRepo:      billRepo,
		analyticsSvc:  analyticsSvc,
		billSvc:       billSvc,
		clock:         timeutil.New(time.UTC),
	}
}

// WithClock pins the calendar used for month windows.
func (s *DerivedAnalyticsService) WithClock(c *timeutil.Clock) *DerivedAnalyticsService {
	s.clock = c
	return s
}

// GetMonthlyAnomalies scans every category's monthly spending in one currency
// over the last `months` months (including the current month) with a Hampel
// filter. Each flagged month carries a human-readable explanation naming the
// amount, the local median, and the filter parameters. Sufficient is false
// when months < MinAnomalyMonths; the data is still returned.
func (s *DerivedAnalyticsService) GetMonthlyAnomalies(ctx context.Context, userEmail, currency string, months int) (*domain.AnomalyReport, error) {
	if currency == "" {
		return nil, fmt.Errorf("currency is required")
	}
	if months < 1 || months > 24 {
		return nil, fmt.Errorf("months must be between 1 and 24")
	}

	now := s.clock.Now()
	currentMonth := now.Format("2006-01")
	fromMonth := addMonths(currentMonth, -(months - 1))
	from, err := s.clock.ParseDate(fromMonth + "-01")
	if err != nil {
		return nil, fmt.Errorf("anomaly from: %w", err)
	}
	_, to, err := s.clock.MonthRange(addMonths(currentMonth, 1))
	if err != nil {
		return nil, fmt.Errorf("anomaly to: %w", err)
	}

	rows, err := s.analyticsRepo.GetCategoryMonthlySpendAll(ctx, userEmail, from, to)
	if err != nil {
		return nil, fmt.Errorf("category monthly spend all: %w", err)
	}

	// Group rows into per-category zero-filled series.
	byCategory := make(map[string][]int64)
	for _, row := range rows {
		if row.Currency != currency {
			continue
		}
		if _, ok := byCategory[row.Category]; !ok {
			byCategory[row.Category] = make([]int64, months)
		}
		idx := monthIndex(fromMonth, row.Month, months)
		if idx >= 0 {
			byCategory[row.Category][idx] = row.AmountCents
		}
	}

	categories := make([]string, 0, len(byCategory))
	for cat := range byCategory {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	report := &domain.AnomalyReport{
		Currency:   currency,
		Months:     months,
		Sufficient: months >= domain.MinAnomalyMonths,
		Anomalies:  []domain.Anomaly{},
		Assumptions: []string{
			fmt.Sprintf("anomalies use a Hampel filter: rolling median + MAD, window %d months, n_sigma %.1f", domain.HampelWindow, domain.HampelNSigma),
			fmt.Sprintf("an anomaly scan requires at least %d months of history; shorter windows are not presented as anomalies", domain.MinAnomalyMonths),
		},
	}

	for _, cat := range categories {
		series := byCategory[cat]
		for _, idx := range domain.HampelOutliers(series, domain.HampelWindow, domain.HampelNSigma) {
			amount := series[idx]
			median := domain.LocalMedian(series, idx, domain.HampelWindow)
			ratio := 0.0
			if median > 0 {
				ratio = float64(amount) / median
			}
			month := addMonths(fromMonth, idx)
			report.Anomalies = append(report.Anomalies, domain.Anomaly{
				Category:    cat,
				Currency:    currency,
				Month:       month,
				AmountCents: amount,
				MedianCents: int64(median),
				Ratio:       ratio,
				Explanation: domain.AnomalyExplanation(cat, currency, month, amount, int64(median), ratio, domain.HampelWindow, domain.HampelNSigma),
			})
		}
	}

	return report, nil
}

// monthIndex maps a YYYY-MM month to its offset within a window starting at
// fromMonth; -1 when the month is outside the window.
func monthIndex(fromMonth, month string, months int) int {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return -1
	}
	base, err := time.Parse("2006-01", fromMonth)
	if err != nil {
		return -1
	}
	diff := int(t.Year()*12+int(t.Month())) - int(base.Year()*12+int(base.Month()))
	if diff < 0 || diff >= months {
		return -1
	}
	return diff
}
