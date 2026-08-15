package application

import (
	"testing"

	"github.com/jjspscl/my/internal/contexts/intelligence/domain"
)

func TestConfidenceCalibrationCaps(t *testing.T) {
	svc := NewConfidenceService()

	cases := []struct {
		name     string
		field    string
		evidence []domain.Evidence
		want     float64
	}{
		{
			name:     "category model only stays below preselect",
			field:    domain.FieldCategory,
			evidence: []domain.Evidence{{Source: domain.EvidenceModel}},
			want:     0.70,
		},
		{
			name:  "category model plus web capped at 0.85",
			field: domain.FieldCategory,
			evidence: []domain.Evidence{
				{Source: domain.EvidenceModel},
				{Source: domain.EvidenceWeb},
			},
			want: 0.85,
		},
		{
			name:  "category user rule reaches preselect",
			field: domain.FieldCategory,
			evidence: []domain.Evidence{
				{Source: domain.EvidenceUserRule},
			},
			want: 0.98,
		},
		{
			name:  "category history + model never exceeds history cap",
			field: domain.FieldCategory,
			evidence: []domain.Evidence{
				{Source: domain.EvidenceModel},
				{Source: domain.EvidenceHistory},
			},
			want: 0.93, // 0.90 + 0.03 corroboration bonus
		},
		{
			name:  "transfer model only never preselects",
			field: domain.FieldTransfer,
			evidence: []domain.Evidence{
				{Source: domain.EvidenceModel},
			},
			want: 0.59,
		},
		{
			name:  "transfer user rule preselects",
			field: domain.FieldTransfer,
			evidence: []domain.Evidence{
				{Source: domain.EvidenceUserRule},
			},
			want: 0.98,
		},
		{
			name:     "no evidence scores zero",
			field:    domain.FieldCategory,
			evidence: nil,
			want:     0,
		},
		{
			name:  "duplicate sources count once",
			field: domain.FieldMerchant,
			evidence: []domain.Evidence{
				{Source: domain.EvidenceModel},
				{Source: domain.EvidenceModel},
				{Source: domain.EvidenceModel},
			},
			want: 0.70,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := svc.Calibrate(c.field, c.evidence)
			if got != c.want {
				t.Errorf("Calibrate(%s) = %v, want %v", c.field, got, c.want)
			}
		})
	}
}

func TestStatusFor(t *testing.T) {
	if got := StatusFor(0.98); got != "preselect" {
		t.Errorf("0.98 -> %q, want preselect", got)
	}
	if got := StatusFor(0.90); got != "preselect" {
		t.Errorf("0.90 -> %q, want preselect", got)
	}
	if got := StatusFor(0.60); got != "review" {
		t.Errorf("0.60 -> %q, want review", got)
	}
	if got := StatusFor(0.59); got != "unresolved" {
		t.Errorf("0.59 -> %q, want unresolved", got)
	}
}
