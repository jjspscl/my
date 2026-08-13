package domain

import (
	"fmt"
	"time"
)

// Classification is the essential-vs-discretionary taxonomy applied to
// categories. Analytics that depend on it must treat a missing or
// unclassified category as non-essential and report the unclassified share.
type Classification string

const (
	ClassificationNeeds        Classification = "needs"
	ClassificationWants        Classification = "wants"
	ClassificationSavings      Classification = "savings"
	ClassificationIncome       Classification = "income"
	ClassificationDebt         Classification = "debt"
	ClassificationOther        Classification = "other"
	ClassificationUnclassified Classification = "unclassified"
)

// ValidClassification reports whether c is one of the allowed values.
func ValidClassification(c Classification) bool {
	switch c {
	case ClassificationNeeds, ClassificationWants, ClassificationSavings,
		ClassificationIncome, ClassificationDebt, ClassificationOther,
		ClassificationUnclassified:
		return true
	}
	return false
}

// Category is the metadata row for one transaction category. The name is the
// primary key and matches transactions.category as free text; there is no
// foreign key so free-text entry and offline replay keep working.
type Category struct {
	Name           string
	Classification Classification
	Essential      bool
	Active         bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewCategory validates the classification and returns a category ready to
// persist.
func NewCategory(name string, classification Classification, essential, active bool) (*Category, error) {
	if name == "" {
		return nil, fmt.Errorf("category name is required")
	}
	if !ValidClassification(classification) {
		return nil, fmt.Errorf("invalid classification %q", classification)
	}
	now := time.Now().UTC()
	return &Category{
		Name:           name,
		Classification: classification,
		Essential:      essential,
		Active:         active,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}
