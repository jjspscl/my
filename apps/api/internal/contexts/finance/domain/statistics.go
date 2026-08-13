package domain

import (
	"math"
	"sort"
)

// Robust statistics for time-series anomaly detection. The Hampel filter is
// the standard robust alternative to the three-sigma rule for time series: it
// compares each point against the median of a rolling window, scaled by the
// window's median absolute deviation (MAD), so a genuine trend does not
// inflate the baseline the way a global mean would.
//
// Constants follow the published defaults: the MAD scale factor 1.4826 makes
// MAD a consistent estimator of sigma at the normal, and n_sigma = 3.0 is the
// Hampel default used by MATLAB and the reference Python implementations.

const (
	// MADScaleFactor converts MAD to an estimate of sigma at the normal
	// (1 / Φ⁻¹(0.75)).
	MADScaleFactor = 1.4826
	// MeanADScaleFactor is the equivalent scale for the mean absolute
	// deviation, used when MAD is zero (a window with no spread).
	MeanADScaleFactor = 1.253314
	// HampelNSigma is the default outlier threshold in sigma units.
	HampelNSigma = 3.0
	// HampelWindow is the default rolling window size in months.
	HampelWindow = 5
	// MinAnomalyMonths is the minimum sample before anomalies are reported as
	// sufficient. Shorter samples are returned with Sufficient=false.
	MinAnomalyMonths = 6
)

// Median returns the median of values as a float64 (the median of an even
// count is the mean of the two middle values). An empty slice returns 0.
func Median(values []int64) float64 {
	n := len(values)
	if n == 0 {
		return 0
	}
	sorted := make([]int64, n)
	copy(sorted, values)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	if n%2 == 1 {
		return float64(sorted[n/2])
	}
	return (float64(sorted[n/2-1]) + float64(sorted[n/2])) / 2
}

// MedianF is the float64 variant of Median.
func MedianF(values []float64) float64 {
	n := len(values)
	if n == 0 {
		return 0
	}
	sorted := make([]float64, n)
	copy(sorted, values)
	sort.Float64s(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

// MAD returns the median absolute deviation from the median. It is robust to
// up to 50% contamination, unlike the standard deviation.
func MAD(values []int64) float64 {
	med := Median(values)
	devs := make([]float64, len(values))
	for i, v := range values {
		devs[i] = math.Abs(float64(v) - med)
	}
	return MedianF(devs)
}

// MeanAD returns the mean absolute deviation from the median. It is the
// documented fallback when MAD is zero (a window whose values are all equal to
// the median), scaled by MeanADScaleFactor.
func MeanAD(values []int64) float64 {
	n := len(values)
	if n == 0 {
		return 0
	}
	med := Median(values)
	var sum float64
	for _, v := range values {
		sum += math.Abs(float64(v) - med)
	}
	return sum / float64(n)
}

// LocalMedian returns the median of the window of size `window` centered on
// index i, truncated at the edges (a trailing window when the centered one
// would leave the slice). It is used to build the human-readable explanation
// for an anomaly so the number shown matches the filter's decision.
func LocalMedian(series []int64, i, window int) float64 {
	lo, hi := windowBounds(len(series), i, window)
	return Median(series[lo:hi])
}

// HampelOutliers returns the indices of series values that deviate from the
// local window median by more than nSigma * MADScaleFactor * MAD. A window
// whose MAD is zero is a constant baseline; deviations from it are not flagged
// (a category that spent the same amount for months and then moved slightly is
// not news). The window is centered and truncated at the edges, so the first
// and last points use a trailing window.
func HampelOutliers(series []int64, window int, nSigma float64) []int {
	n := len(series)
	if n == 0 || window < 3 {
		return nil
	}
	var out []int
	for i := 0; i < n; i++ {
		lo, hi := windowBounds(n, i, window)
		win := series[lo:hi]
		med := Median(win)
		scale := MAD(win) * MADScaleFactor
		if scale == 0 {
			continue
		}
		if math.Abs(float64(series[i])-med) > nSigma*scale {
			out = append(out, i)
		}
	}
	return out
}

// windowBounds returns the half-open [lo, hi) window of size `window` centered
// at index i, clamped to the slice.
func windowBounds(n, i, window int) (int, int) {
	half := window / 2
	lo := i - half
	if lo < 0 {
		lo = 0
	}
	hi := i + half + 1
	if hi > n {
		hi = n
	}
	return lo, hi
}
