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

// monthIndex maps a YYYY-MM string to its offset within a window starting at
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

// GetRecurringCharges scans expense over the last `months` months and groups
// it by category. A category is recurring when it has at least
// RecurringMinOccurrences charges across at least RecurringMinDistinctMonths
// distinct months. Each recurring charge is classified against explicit bills
// as tracked / untracked / amount_changed. Cadence beats amount: amount
// similarity is not a membership requirement, only what separates tracked from
// amount_changed.
func (s *DerivedAnalyticsService) GetRecurringCharges(ctx context.Context, userEmail, currency string, months int) (*domain.RecurringChargesSummary, error) {
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
		return nil, fmt.Errorf("recurring from: %w", err)
	}
	_, to, err := s.clock.MonthRange(addMonths(currentMonth, 1))
	if err != nil {
		return nil, fmt.Errorf("recurring to: %w", err)
	}

	amounts, err := s.analyticsRepo.GetExpenseAmounts(ctx, userEmail, from, to)
	if err != nil {
		return nil, fmt.Errorf("expense amounts: %w", err)
	}

	// Group by category within the requested currency.
	type group struct {
		amounts []int64
		months  map[string]bool
	}
	groups := make(map[string]*group)
	for _, a := range amounts {
		if a.Currency != currency {
			continue
		}
		g, ok := groups[a.Category]
		if !ok {
			g = &group{months: make(map[string]bool)}
			groups[a.Category] = g
		}
		g.amounts = append(g.amounts, a.AmountCents)
		g.months[a.Month] = true
	}

	bills, err := s.billRepo.ListBills(ctx, userEmail)
	if err != nil {
		return nil, fmt.Errorf("list bills: %w", err)
	}
	billByCategory := make(map[string]*domain.RecurringBill)
	for _, b := range bills {
		if _, ok := billByCategory[b.Category]; !ok {
			billByCategory[b.Category] = b
		}
	}

	categories := make([]string, 0, len(groups))
	for cat := range groups {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	summary := &domain.RecurringChargesSummary{
		Currency: currency,
		Months:   months,
		Charges:  []domain.RecurringCharge{},
		Assumptions: []string{
			fmt.Sprintf("a charge is recurring with at least %d occurrences across at least %d distinct months", domain.RecurringMinOccurrences, domain.RecurringMinDistinctMonths),
			fmt.Sprintf("amount tolerance for tracked status: ±%.0f%% or ±%s, whichever is larger", domain.RecurringTolerancePct*100, domain.FormatMinor(domain.RecurringToleranceFloorCents, currency)),
			"cadence beats amount: a charge that recurs on a predictable rhythm is recurring even when the amount drifts",
			"no charge is claimed to be unused; usage data does not exist",
		},
	}

	for _, cat := range categories {
		g := groups[cat]
		if len(g.amounts) < domain.RecurringMinOccurrences || len(g.months) < domain.RecurringMinDistinctMonths {
			continue
		}
		median := int64(domain.Median(g.amounts))
		bill := billByCategory[cat]

		charge := domain.RecurringCharge{
			Category:       cat,
			Currency:       currency,
			Occurrences:    len(g.amounts),
			DistinctMonths: len(g.months),
			MedianCents:    median,
		}

		switch {
		case bill == nil:
			charge.Status = domain.RecurringChargeUntracked
			charge.Explanation = fmt.Sprintf(
				"%s recurs %d times across %d months (median %s) with no matching bill",
				cat, len(g.amounts), len(g.months), domain.FormatMinor(median, currency),
			)
		case withinRecurringTolerance(median, bill.AmountCents):
			charge.Status = domain.RecurringChargeTracked
			charge.BillName = bill.Name
			charge.BillAmountCents = bill.AmountCents
			charge.Explanation = fmt.Sprintf(
				"%s matches bill %q (%s/mo); median charge %s is within tolerance",
				cat, bill.Name, domain.FormatMinor(bill.AmountCents, currency), domain.FormatMinor(median, currency),
			)
		default:
			charge.Status = domain.RecurringChargeAmountChanged
			charge.BillName = bill.Name
			charge.BillAmountCents = bill.AmountCents
			diffPct := 0.0
			if bill.AmountCents > 0 {
				diffPct = (float64(median-bill.AmountCents) / float64(bill.AmountCents)) * 100
			}
			charge.Explanation = fmt.Sprintf(
				"%s median charge %s differs from bill '%s' (%s/mo) by %.1f%%, beyond the ±%.0f%% tolerance",
				cat, domain.FormatMinor(median, currency), bill.Name, domain.FormatMinor(bill.AmountCents, currency),
				diffPct, domain.RecurringTolerancePct*100,
			)
		}
		summary.Charges = append(summary.Charges, charge)
	}

	return summary, nil
}

// withinRecurringTolerance reports whether median is within the recurring
// tolerance of the bill amount: ±RecurringTolerancePct or
// ±RecurringToleranceFloorCents, whichever is larger.
func withinRecurringTolerance(median, billAmount int64) bool {
	diff := median - billAmount
	if diff < 0 {
		diff = -diff
	}
	floor := int64(float64(billAmount) * domain.RecurringTolerancePct)
	if floor < domain.RecurringToleranceFloorCents {
		floor = domain.RecurringToleranceFloorCents
	}
	return diff <= floor
}

// GetBillReconciliation compares each bill's expected amount against what was
// actually paid in one month. Expected is the bill amount times the number of
// occurrences in the month; variance is paid minus expected. Paid occurrences
// without a linked transaction are counted separately.
func (s *DerivedAnalyticsService) GetBillReconciliation(ctx context.Context, userEmail, month string) (*domain.BillReconciliation, error) {
	if month == "" {
		return nil, fmt.Errorf("month is required")
	}
	from, to, err := s.clock.MonthRange(month)
	if err != nil {
		return nil, fmt.Errorf("month range: %w", err)
	}

	rows, err := s.analyticsRepo.GetBillReconciliation(ctx, userEmail, from, to)
	if err != nil {
		return nil, fmt.Errorf("bill reconciliation: %w", err)
	}

	recon := &domain.BillReconciliation{
		Month: month,
		Items: []domain.BillReconciliationItem{},
		Assumptions: []string{
			"expected = bill amount × occurrences in the month",
			"variance = paid − expected",
			"paid occurrences without a linked transaction are counted separately",
		},
	}

	for _, row := range rows {
		occurrences := occurrencesInMonth(row, from, to)
		expected := row.AmountCents * int64(occurrences)
		item := domain.BillReconciliationItem{
			BillID:                      row.BillID,
			Name:                        row.Name,
			Category:                    row.Category,
			Currency:                    row.Currency,
			ExpectedCents:               expected,
			PaidCents:                   row.PaidCents,
			VarianceCents:               row.PaidCents - expected,
			PaidCount:                   row.PaidCount,
			PaidWithoutTransactionCount: row.PaidWithoutTransactionCount,
		}
		item.Explanation = fmt.Sprintf(
			"%s expected %s (%d occurrence(s) in %s), paid %s, variance %s",
			row.Name, domain.FormatMinor(expected, row.Currency), occurrences, month,
			domain.FormatMinor(row.PaidCents, row.Currency), domain.FormatMinor(item.VarianceCents, row.Currency),
		)
		if row.PaidWithoutTransactionCount > 0 {
			item.Explanation += fmt.Sprintf("; %d paid occurrence(s) have no linked transaction", row.PaidWithoutTransactionCount)
		}
		recon.Items = append(recon.Items, item)
	}

	return recon, nil
}

// occurrencesInMonth returns how many times a bill recurs within [from, to).
// Monthly bills recur once; weekly bills recur every 7 days from their start
// date; yearly bills recur once in the month of their start date.
func occurrencesInMonth(row domain.BillReconciliationRow, from, to time.Time) int {
	switch row.Frequency {
	case domain.FrequencyWeekly:
		count := 0
		start := row.StartDate
		if start.Before(from) {
			start = from
		}
		for d := start; d.Before(to); d = d.AddDate(0, 0, 7) {
			count++
		}
		return count
	case domain.FrequencyYearly:
		// A yearly bill recurs every year in its start month.
		if int(row.StartDate.Month()) == int(from.Month()) {
			return 1
		}
		return 0
	default: // monthly
		return 1
	}
}
