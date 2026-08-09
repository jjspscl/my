package domain

import "fmt"

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
