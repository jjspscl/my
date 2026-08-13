package domain

import (
	"math"
	"strings"
	"testing"
)

func TestMedian(t *testing.T) {
	cases := []struct {
		name   string
		values []int64
		want   float64
	}{
		{name: "empty", values: nil, want: 0},
		{name: "single", values: []int64{7}, want: 7},
		{name: "odd count", values: []int64{3, 1, 2}, want: 2},
		{name: "even count", values: []int64{1, 2, 3, 4}, want: 2.5},
		{name: "unsorted", values: []int64{9, 4, 7, 4, 2}, want: 4},
		{name: "duplicates", values: []int64{5, 5, 5}, want: 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Median(tc.values); got != tc.want {
				t.Errorf("Median(%v) = %v, want %v", tc.values, got, tc.want)
			}
		})
	}
}

func TestMAD(t *testing.T) {
	// {1, 2, 3, 4, 100}: median 3, absolute deviations {2,1,0,1,97},
	// median of deviations = 1.
	got := MAD([]int64{1, 2, 3, 4, 100})
	if got != 1 {
		t.Errorf("MAD = %v, want 1", got)
	}
	// All-equal series has MAD 0.
	if got := MAD([]int64{500, 500, 500}); got != 0 {
		t.Errorf("MAD of constant series = %v, want 0", got)
	}
	if got := MAD(nil); got != 0 {
		t.Errorf("MAD of empty series = %v, want 0", got)
	}
}

func TestMeanAD(t *testing.T) {
	// {1, 2, 3}: median 2, deviations {1,0,1}, mean = 2/3.
	got := MeanAD([]int64{1, 2, 3})
	if math.Abs(got-2.0/3.0) > 1e-9 {
		t.Errorf("MeanAD = %v, want %v", got, 2.0/3.0)
	}
	if got := MeanAD(nil); got != 0 {
		t.Errorf("MeanAD of empty series = %v, want 0", got)
	}
}

func TestHampelOutliers(t *testing.T) {
	t.Run("flags clear spike", func(t *testing.T) {
		// Alternating baseline with one 10x spike. The window around the spike
		// has nonzero MAD, so the spike is flagged.
		series := []int64{4500, 5500, 4500, 5500, 4500, 5500, 50000, 4500, 5500, 4500, 5500, 4500}
		out := HampelOutliers(series, HampelWindow, HampelNSigma)
		if len(out) != 1 || out[0] != 6 {
			t.Fatalf("outliers = %v, want [6]", out)
		}
	})

	t.Run("stable series flags nothing", func(t *testing.T) {
		series := make([]int64, 12)
		for i := range series {
			series[i] = 1000 + int64(i)*100 // gentle trend, no spike
		}
		out := HampelOutliers(series, HampelWindow, HampelNSigma)
		if len(out) != 0 {
			t.Fatalf("outliers = %v, want none", out)
		}
	})

	t.Run("constant baseline flags nothing", func(t *testing.T) {
		// MAD is zero across every window: a constant baseline. A small
		// deviation is not news and must not be flagged.
		series := []int64{500, 500, 500, 500, 500, 500, 501, 500, 500}
		out := HampelOutliers(series, HampelWindow, HampelNSigma)
		if len(out) != 0 {
			t.Fatalf("outliers = %v, want none (constant baseline)", out)
		}
	})

	t.Run("empty and tiny series return nil", func(t *testing.T) {
		if out := HampelOutliers(nil, HampelWindow, HampelNSigma); out != nil {
			t.Fatalf("empty series outliers = %v, want nil", out)
		}
		if out := HampelOutliers([]int64{1, 2}, HampelWindow, HampelNSigma); out != nil {
			t.Fatalf("tiny series outliers = %v, want nil", out)
		}
	})
}

func TestLocalMedian(t *testing.T) {
	// Window 5 centered on index 2 of {1,2,3,4,5} → {1,2,3,4,5} → 3.
	got := LocalMedian([]int64{1, 2, 3, 4, 5}, 2, 5)
	if got != 3 {
		t.Errorf("LocalMedian = %v, want 3", got)
	}
	// Edge: index 0 of the same series → trailing window {1,2,3} → 2.
	if got := LocalMedian([]int64{1, 2, 3, 4, 5}, 0, 5); got != 2 {
		t.Errorf("edge LocalMedian = %v, want 2", got)
	}
}

func TestFormatMinor(t *testing.T) {
	cases := []struct {
		cents    int64
		currency string
		want     string
	}{
		{420000, "PHP", "₱4,200.00"},
		{0, "PHP", "₱0.00"},
		{99, "PHP", "₱0.99"},
		{123456789, "PHP", "₱1,234,567.89"},
		{500, "USD", "$5.00"},
		{-250, "PHP", "-₱2.50"},
		{123456, "XYZ", "XYZ 1,234.56"},
	}
	for _, tc := range cases {
		if got := FormatMinor(tc.cents, tc.currency); got != tc.want {
			t.Errorf("FormatMinor(%d, %s) = %q, want %q", tc.cents, tc.currency, got, tc.want)
		}
	}
}

func TestAnomalyExplanation(t *testing.T) {
	withMedian := AnomalyExplanation("Food", "PHP", "2026-07", 420000, 140000, 3.0, 5, 3.0)
	want := "Food spending in 2026-07 was ₱4,200.00, about 3.0x the local median of ₱1,400.00 (Hampel filter, window 5 months, n_sigma 3.0)"
	if withMedian != want {
		t.Errorf("explanation = %q, want %q", withMedian, want)
	}

	noMedian := AnomalyExplanation("Food", "PHP", "2026-07", 420000, 0, 0, 5, 3.0)
	if noMedian == "" || strings.Contains(noMedian, "3.0x") {
		t.Errorf("no-median explanation should omit the ratio: %q", noMedian)
	}
}
