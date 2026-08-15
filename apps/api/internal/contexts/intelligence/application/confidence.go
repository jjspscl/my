package application

import (
	"github.com/jjspscl/my/internal/contexts/intelligence/domain"
)

// ConfidenceService calibrates per-field confidence from evidence.
//
// Model self-rating is NOT trustworthy alone: every field kind has a ceiling
// based on which evidence sources corroborate the claim. High-confidence
// preselection (>= 0.90) therefore requires more than a model guess.
type ConfidenceService struct{}

func NewConfidenceService() *ConfidenceService {
	return &ConfidenceService{}
}

// Thresholds shared by the UI and this service.
const (
	PreselectThreshold = 0.90 // >= 0.90: preselected, still editable
	ReviewThreshold    = 0.60 // 0.60-0.89: requires review; < 0.60: unresolved
)

// fieldCaps returns the maximum confidence a field kind may reach for a given
// set of evidence sources.
func fieldCaps(field string) map[domain.EvidenceSource]float64 {
	switch field {
	case domain.FieldCategory:
		return map[domain.EvidenceSource]float64{
			domain.EvidenceUserRule: 0.98,
			domain.EvidenceHistory:  0.90,
			domain.EvidenceWeb:      0.85,
			domain.EvidenceModel:    0.70,
		}
	case domain.FieldMerchant:
		return map[domain.EvidenceSource]float64{
			domain.EvidenceUserRule: 0.98,
			domain.EvidenceHistory:  0.90,
			domain.EvidenceWeb:      0.85,
			domain.EvidenceModel:    0.70,
		}
	case domain.FieldRelationship:
		return map[domain.EvidenceSource]float64{
			domain.EvidenceUserRule: 0.98,
			domain.EvidenceHistory:  0.90,
			domain.EvidenceModel:    0.70,
		}
	case domain.FieldTransfer:
		// Ownership of a counterparty wallet can never be confirmed by a model
		// or the web alone; only an explicit user mapping or prior history of
		// the same mapping reaches preselect territory.
		return map[domain.EvidenceSource]float64{
			domain.EvidenceUserRule: 0.98,
			domain.EvidenceHistory:  0.85,
			domain.EvidenceModel:    0.59, // never preselects; requires review
		}
	default:
		return map[domain.EvidenceSource]float64{}
	}
}

// Calibrate computes the confidence for one field suggestion from its
// evidence. It starts at zero and takes the best single corroborating source
// (bounded by that source's cap), then adds a small bonus for additional
// independent sources — never exceeding the strongest cap.
func (s *ConfidenceService) Calibrate(field string, evidence []domain.Evidence) float64 {
	caps := fieldCaps(field)
	if len(caps) == 0 {
		return 0
	}

	seen := map[domain.EvidenceSource]bool{}
	best := 0.0
	for _, e := range evidence {
		if seen[e.Source] {
			continue
		}
		seen[e.Source] = true
		if cap := caps[e.Source]; cap > best {
			best = cap
		}
	}

	// Corroboration bonus: a trusted source (user rule or history) combined
	// with an independent source earns a small bump, capped by the field's
	// strongest ceiling. Web results are untrusted and never earn the bump.
	hasTrusted := seen[domain.EvidenceUserRule] || seen[domain.EvidenceHistory]
	if hasTrusted && len(seen) >= 2 {
		best += 0.03
	}
	if best > strongestCap(caps) {
		best = strongestCap(caps)
	}
	return round3(best)
}

func strongestCap(caps map[domain.EvidenceSource]float64) float64 {
	strongest := 0.0
	for _, c := range caps {
		if c > strongest {
			strongest = c
		}
	}
	return strongest
}

func round3(v float64) float64 {
	return float64(int(v*1000+0.5)) / 1000
}

// StatusFor maps a calibrated score to the shared UX bucket.
func StatusFor(score float64) string {
	switch {
	case score >= PreselectThreshold:
		return "preselect"
	case score >= ReviewThreshold:
		return "review"
	default:
		return "unresolved"
	}
}
