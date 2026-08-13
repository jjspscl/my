package application

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
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

// checkClassification refuses analytics whose essential-spend consumers depend
// on classification when more than MaxUnclassifiedShare of spending in the
// currency is unclassified over [from, to).
func (s *DerivedAnalyticsService) checkClassification(ctx context.Context, userEmail, currency string, from, to time.Time) error {
	splits, err := s.analyticsRepo.GetUnclassifiedSpending(ctx, userEmail, from, to)
	if err != nil {
		return fmt.Errorf("unclassified spending: %w", err)
	}
	for _, u := range splits {
		if u.Currency != currency || u.TotalCents <= 0 {
			continue
		}
		share := (float64(u.UnclassifiedCents) / float64(u.TotalCents)) * 100
		if share > domain.MaxUnclassifiedShare*100 {
			top, err := s.analyticsRepo.GetTopUnclassifiedCategories(ctx, userEmail, from, to, 5)
			if err != nil {
				return fmt.Errorf("top unclassified categories: %w", err)
			}
			names := make([]string, 0, len(top))
			for _, c := range top {
				names = append(names, c.Category)
			}
			return &domain.ErrInsufficientClassification{
				UnclassifiedSharePct: share,
				TopUnclassified:      names,
			}
		}
	}
	return nil
}

// essentialWindow returns the half-open range covering the last
// EmergencyFundWindowMonths months (including the current month) plus the
// window's first month string for series indexing.
func (s *DerivedAnalyticsService) essentialWindow() (time.Time, time.Time, string, error) {
	now := s.clock.Now()
	currentMonth := now.Format("2006-01")
	fromMonth := addMonths(currentMonth, -(domain.EmergencyFundWindowMonths - 1))
	from, err := s.clock.ParseDate(fromMonth + "-01")
	if err != nil {
		return time.Time{}, time.Time{}, "", fmt.Errorf("essential window from: %w", err)
	}
	_, to, err := s.clock.MonthRange(addMonths(currentMonth, 1))
	if err != nil {
		return time.Time{}, time.Time{}, "", fmt.Errorf("essential window to: %w", err)
	}
	return from, to, fromMonth, nil
}

// medianEssentialSpend returns the median monthly essential spend for one
// currency over the window, zero-filled so months without essential spending
// count as low spend.
func (s *DerivedAnalyticsService) medianEssentialSpend(ctx context.Context, userEmail, currency string, from, to time.Time, fromMonth string) (int64, error) {
	rows, err := s.analyticsRepo.GetEssentialMonthlySpend(ctx, userEmail, from, to)
	if err != nil {
		return 0, fmt.Errorf("essential monthly spend: %w", err)
	}
	series := make([]int64, domain.EmergencyFundWindowMonths)
	for _, row := range rows {
		if row.Currency != currency {
			continue
		}
		if idx := monthIndex(fromMonth, row.Month, domain.EmergencyFundWindowMonths); idx >= 0 {
			series[idx] = row.AmountCents
		}
	}
	return int64(domain.Median(series)), nil
}

// liquidBalance returns the sum of wallet balances for one currency.
func (s *DerivedAnalyticsService) liquidBalance(ctx context.Context, userEmail, currency string) (int64, error) {
	balances, err := s.walletRepo.GetBalancesByUser(ctx, userEmail)
	if err != nil {
		return 0, fmt.Errorf("wallet balances: %w", err)
	}
	var total int64
	for _, b := range balances {
		if b.Wallet.Currency == currency {
			total += b.BalanceCents
		}
	}
	return total, nil
}

// GetEmergencyFund reports liquid balance against a target range of months of
// essential spending. The default target is 3–6 months (the CFPB/FINRA/Fidelity
// consensus); targetMonths overrides both ends when non-zero. Refuses with
// ErrInsufficientClassification when classification is untrustworthy.
func (s *DerivedAnalyticsService) GetEmergencyFund(ctx context.Context, userEmail, currency string, targetMonths int) (*domain.EmergencyFundStatus, error) {
	if currency == "" {
		return nil, fmt.Errorf("currency is required")
	}
	if targetMonths < 0 || targetMonths > 12 {
		return nil, fmt.Errorf("targetMonths must be between 1 and 12")
	}

	from, to, fromMonth, err := s.essentialWindow()
	if err != nil {
		return nil, err
	}
	if err := s.checkClassification(ctx, userEmail, currency, from, to); err != nil {
		return nil, err
	}

	monthly, err := s.medianEssentialSpend(ctx, userEmail, currency, from, to, fromMonth)
	if err != nil {
		return nil, err
	}
	liquid, err := s.liquidBalance(ctx, userEmail, currency)
	if err != nil {
		return nil, err
	}

	target := [2]int{domain.EmergencyFundMinMonths, domain.EmergencyFundMaxMonths}
	if targetMonths > 0 {
		target = [2]int{targetMonths, targetMonths}
	}

	runway := 0.0
	if monthly > 0 {
		runway = float64(liquid) / float64(monthly)
	}
	shortfallMin := int64(target[0])*monthly - liquid
	if shortfallMin < 0 {
		shortfallMin = 0
	}
	shortfallMax := int64(target[1])*monthly - liquid
	if shortfallMax < 0 {
		shortfallMax = 0
	}

	return &domain.EmergencyFundStatus{
		Currency:              currency,
		LiquidBalanceCents:    liquid,
		MonthlyEssentialCents: monthly,
		MonthsOfRunway:        runway,
		TargetRangeMonths:     target,
		ShortfallToMinCents:   shortfallMin,
		ShortfallToMaxCents:   shortfallMax,
		Assumptions: []string{
			fmt.Sprintf("emergency fund target is %d–%d months of essential spending (CFPB/FINRA/Fidelity consensus)", target[0], target[1]),
			fmt.Sprintf("monthly essential spend is the median over the last %d months in essential categories", domain.EmergencyFundWindowMonths),
			"liquid balance is the sum of wallet balances per currency; it is not net worth",
		},
	}, nil
}

// GetAffordability models a prospective purchase: runway (liquid balance /
// monthly obligation) before and after the purchase. It never returns a
// yes/no; the caller decides from the model and its named assumptions.
func (s *DerivedAnalyticsService) GetAffordability(ctx context.Context, userEmail, currency string, amountCents int64) (*domain.Affordability, error) {
	if currency == "" {
		return nil, fmt.Errorf("currency is required")
	}
	if amountCents <= 0 {
		return nil, fmt.Errorf("amountCents must be positive")
	}

	from, to, fromMonth, err := s.essentialWindow()
	if err != nil {
		return nil, err
	}
	if err := s.checkClassification(ctx, userEmail, currency, from, to); err != nil {
		return nil, err
	}

	monthly, err := s.medianEssentialSpend(ctx, userEmail, currency, from, to, fromMonth)
	if err != nil {
		return nil, err
	}
	liquid, err := s.liquidBalance(ctx, userEmail, currency)
	if err != nil {
		return nil, err
	}

	// Bills due in the next 30 days that are not yet paid.
	upcoming, err := s.billSvc.GetUpcoming(ctx, userEmail, 30)
	if err != nil {
		return nil, fmt.Errorf("upcoming bills: %w", err)
	}
	now := s.clock.TodayStart()
	cutoff := now.AddDate(0, 0, 30)
	var upcomingBills int64
	for _, u := range upcoming {
		if u.Bill.Currency != currency || u.Status == domain.OccurrencePaid {
			continue
		}
		if !u.DueDate.Before(now) && !u.DueDate.After(cutoff) {
			upcomingBills += u.Bill.AmountCents
		}
	}

	obligation := monthly + upcomingBills
	runwayBefore := 0.0
	runwayAfter := 0.0
	if obligation > 0 {
		runwayBefore = float64(liquid) / float64(obligation)
		runwayAfter = float64(liquid-amountCents) / float64(obligation)
	}

	return &domain.Affordability{
		Currency:               currency,
		AmountCents:            amountCents,
		LiquidBalanceCents:     liquid,
		MonthlyEssentialCents:  monthly,
		UpcomingBillsCents:     upcomingBills,
		MonthlyObligationCents: obligation,
		RunwayMonthsBefore:     runwayBefore,
		RunwayMonthsAfter:      runwayAfter,
		Assumptions: []string{
			"affordability is a financial model, not a recommendation; no yes/no is given",
			fmt.Sprintf("monthly obligation = median essential spend (%s) + bills due in the next 30 days (%s)", domain.FormatMinor(monthly, currency), domain.FormatMinor(upcomingBills, currency)),
			"runway = liquid balance / monthly obligation, before and after the purchase",
		},
	}, nil
}

// GetMonthlyDigest composes the monthly summary. Sections that cannot be
// computed (e.g. classification above the unclassified threshold) are omitted
// and named in Omitted; the digest never fails wholesale over one section.
func (s *DerivedAnalyticsService) GetMonthlyDigest(ctx context.Context, userEmail, month string) (*domain.MonthlyDigest, error) {
	if month == "" {
		return nil, fmt.Errorf("month is required")
	}
	from, to, err := s.clock.MonthRange(month)
	if err != nil {
		return nil, fmt.Errorf("month range: %w", err)
	}

	digest := &domain.MonthlyDigest{
		Month:       month,
		Omitted:     []string{},
		Assumptions: []string{"sections that cannot be computed are omitted with the reason, never fabricated"},
	}

	// Cash flow.
	cf, err := s.analyticsSvc.GetCashFlowSummary(ctx, userEmail, from, to)
	if err != nil {
		return nil, fmt.Errorf("cash flow: %w", err)
	}
	digest.CashFlow = domain.DigestCashFlow{Present: true, Currencies: cf.Currencies}
	digest.CashFlow.Summary = digestCashFlowSummary(cf.Currencies)

	// Spending breakdown (graceful degradation on insufficient classification).
	sp, err := s.analyticsSvc.GetSpendingSummary(ctx, userEmail, from, to)
	if err != nil {
		var insufficient *domain.ErrInsufficientClassification
		if errors.As(err, &insufficient) {
			digest.Omitted = append(digest.Omitted, "spending breakdown: "+insufficient.Error())
		} else {
			return nil, fmt.Errorf("spending summary: %w", err)
		}
	} else {
		digest.Spending = domain.DigestSpending{
			Present:              true,
			Currencies:           sp.Currencies,
			UnclassifiedSharePct: sp.UnclassifiedSharePct,
		}
		digest.Spending.Summary = digestSpendingSummary(sp.Currencies)
	}

	// Savings rate.
	rates, err := s.analyticsSvc.GetSavingsRate(ctx, userEmail, from, to)
	if err != nil {
		return nil, fmt.Errorf("savings rate: %w", err)
	}
	digest.SavingsRate = domain.DigestSavings{Present: true, Rates: rates}
	digest.SavingsRate.Summary = digestSavingsSummary(rates)

	// Windowed sections run on the primary currency (first in the cash flow).
	primary := ""
	if len(cf.Currencies) > 0 {
		primary = cf.Currencies[0].Currency
	}
	if primary != "" {
		recurring, err := s.GetRecurringCharges(ctx, userEmail, primary, 6)
		if err != nil {
			return nil, fmt.Errorf("recurring charges: %w", err)
		}
		digest.Recurring = domain.DigestRecurring{Present: true, Charges: recurring.Charges}
		digest.Recurring.Summary = digestRecurringSummary(recurring.Charges)

		anomalies, err := s.GetMonthlyAnomalies(ctx, userEmail, primary, 6)
		if err != nil {
			return nil, fmt.Errorf("anomalies: %w", err)
		}
		digest.Anomalies = domain.DigestAnomalies{Present: true, Anomalies: anomalies.Anomalies}
		digest.Anomalies.Summary = fmt.Sprintf("%d spending anomaly(ies) flagged in %s over the last 6 months", len(anomalies.Anomalies), primary)

		emergency, err := s.GetEmergencyFund(ctx, userEmail, primary, 0)
		if err != nil {
			var insufficient *domain.ErrInsufficientClassification
			if errors.As(err, &insufficient) {
				digest.Omitted = append(digest.Omitted, "emergency fund: "+insufficient.Error())
			} else {
				return nil, fmt.Errorf("emergency fund: %w", err)
			}
		} else {
			digest.Emergency = domain.DigestEmergency{Present: true, Status: emergency}
			digest.Emergency.Summary = fmt.Sprintf(
				"liquid %s covers %.1f months of essential spending (target %d–%d months)",
				domain.FormatMinor(emergency.LiquidBalanceCents, primary), emergency.MonthsOfRunway,
				emergency.TargetRangeMonths[0], emergency.TargetRangeMonths[1],
			)
		}
	} else {
		digest.Omitted = append(digest.Omitted, "recurring charges, anomalies, and emergency fund: no activity in the month")
	}

	return digest, nil
}

func digestCashFlowSummary(currencies []domain.CurrencyCashFlow) string {
	parts := make([]string, 0, len(currencies))
	for _, c := range currencies {
		parts = append(parts, fmt.Sprintf(
			"%s: income %s, expenses %s, net %s",
			c.Currency, domain.FormatMinor(c.IncomeCents, c.Currency),
			domain.FormatMinor(c.ExpenseCents, c.Currency), domain.FormatMinor(c.NetCents, c.Currency),
		))
	}
	return strings.Join(parts, "; ")
}

func digestSpendingSummary(currencies []domain.CurrencySpending) string {
	parts := make([]string, 0, len(currencies))
	for _, c := range currencies {
		parts = append(parts, fmt.Sprintf(
			"%s: needs %s, wants %s, unclassified %s",
			c.Currency,
			domain.FormatMinor(c.ByClassification[domain.ClassificationNeeds], c.Currency),
			domain.FormatMinor(c.ByClassification[domain.ClassificationWants], c.Currency),
			domain.FormatMinor(c.UnclassifiedCents, c.Currency),
		))
	}
	return strings.Join(parts, "; ")
}

func digestSavingsSummary(rates []domain.SavingsRate) string {
	parts := make([]string, 0, len(rates))
	for _, r := range rates {
		if r.ZeroIncome {
			parts = append(parts, r.Currency+": no income in the month")
			continue
		}
		parts = append(parts, fmt.Sprintf("%s: %.1f%%", r.Currency, r.RatePercent))
	}
	return strings.Join(parts, "; ")
}

func digestRecurringSummary(charges []domain.RecurringCharge) string {
	var tracked, changed, untracked int
	for _, c := range charges {
		switch c.Status {
		case domain.RecurringChargeTracked:
			tracked++
		case domain.RecurringChargeAmountChanged:
			changed++
		default:
			untracked++
		}
	}
	return fmt.Sprintf("%d recurring charge(s): %d tracked, %d amount_changed, %d untracked", len(charges), tracked, changed, untracked)
}
