package domain

import "fmt"

// Recurring-charge classification constants. Cadence beats amount: a charge
// that recurs on a predictable rhythm is recurring even when the amount drifts.
const (
	// RecurringMinOccurrences is the minimum number of charges before a
	// category is considered recurring.
	RecurringMinOccurrences = 3
	// RecurringMinDistinctMonths is the minimum number of distinct months the
	// charges must span.
	RecurringMinDistinctMonths = 3
	// RecurringTolerancePct is the amount tolerance before a recurring charge
	// is classified amount_changed instead of tracked. 10% is calibrated to
	// skip normal utility fluctuation.
	RecurringTolerancePct = 0.10
	// RecurringToleranceFloorCents is the absolute tolerance floor so a small
	// subscription does not flip status over a few cents.
	RecurringToleranceFloorCents = 50
)

// RecurringChargeStatus classifies a recurring charge against explicit bills.
type RecurringChargeStatus string

const (
	// RecurringChargeTracked matches a bill within tolerance.
	RecurringChargeTracked RecurringChargeStatus = "tracked"
	// RecurringChargeUntracked has no matching bill.
	RecurringChargeUntracked RecurringChargeStatus = "untracked"
	// RecurringChargeAmountChanged matches a bill but drifts beyond tolerance.
	RecurringChargeAmountChanged RecurringChargeStatus = "amount_changed"
)

// RecurringCharge is one detected recurring expense. The explanation is the
// primary output; the status is derived from it. A charge is never claimed to
// be unused — no usage data exists.
type RecurringCharge struct {
	Category        string
	Currency        string
	Occurrences     int
	DistinctMonths  int
	MedianCents     int64
	Status          RecurringChargeStatus
	BillName        string // set when tracked or amount_changed
	BillAmountCents int64  // set when tracked or amount_changed
	Explanation     string
}

// RecurringChargesSummary is the recurring-charge scan over a window.
type RecurringChargesSummary struct {
	Currency    string
	Months      int
	Charges     []RecurringCharge
	Assumptions []string
}

// BillReconciliationItem is one bill's expected-versus-actual for a month.
// VarianceCents is PaidCents - ExpectedCents. PaidWithoutTransactionCount is
// the number of paid occurrences with no linked transaction.
type BillReconciliationItem struct {
	BillID                      string
	Name                        string
	Category                    string
	Currency                    string
	ExpectedCents               int64
	PaidCents                   int64
	VarianceCents               int64
	PaidCount                   int
	PaidWithoutTransactionCount int
	Explanation                 string
}

// BillReconciliation is the expected-versus-actual comparison for one month.
type BillReconciliation struct {
	Month       string
	Items       []BillReconciliationItem
	Assumptions []string
}

// Anomaly is one flagged month of spending in a category. The explanation is
// the primary output: it states the amount, the local median it was compared
// against, and the filter parameters. No opaque score is exposed.
type Anomaly struct {
	Category    string
	Currency    string
	Month       string // YYYY-MM
	AmountCents int64
	MedianCents int64
	Ratio       float64 // amount / median; 0 when median is 0
	Explanation string
}

// AnomalyReport is the per-currency anomaly scan over a window. Sufficient is
// false when the window is shorter than MinAnomalyMonths; the data is still
// returned but must not be presented as a definitive anomaly list.
type AnomalyReport struct {
	Currency    string
	Months      int
	Sufficient  bool
	Anomalies   []Anomaly
	Assumptions []string
}

// AnomalyExplanation builds the human-readable explanation for one anomaly.
// When the local median is zero (no comparable prior spending in the window)
// the ratio is omitted rather than reported as infinite.
func AnomalyExplanation(category, currency, month string, amountCents, medianCents int64, ratio float64, window int, nSigma float64) string {
	if medianCents <= 0 {
		return fmt.Sprintf(
			"%s spending in %s was %s with no comparable local spending in the prior window (Hampel filter, window %d months, n_sigma %.1f)",
			category, month, FormatMinor(amountCents, currency), window, nSigma,
		)
	}
	return fmt.Sprintf(
		"%s spending in %s was %s, about %.1fx the local median of %s (Hampel filter, window %d months, n_sigma %.1f)",
		category, month, FormatMinor(amountCents, currency), ratio, FormatMinor(medianCents, currency), window, nSigma,
	)
}
