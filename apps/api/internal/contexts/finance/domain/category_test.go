package domain

import "testing"

func TestNewCategoryValidatesName(t *testing.T) {
	if _, err := NewCategory("", ClassificationNeeds, false, true); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestNewCategoryValidatesClassification(t *testing.T) {
	if _, err := NewCategory("Food", "bogus", false, true); err == nil {
		t.Fatal("expected error for invalid classification")
	}
	if _, err := NewCategory("Food", ClassificationUnclassified, false, true); err != nil {
		t.Fatalf("unexpected error for valid classification: %v", err)
	}
}

func TestValidClassification(t *testing.T) {
	valid := []Classification{
		ClassificationNeeds, ClassificationWants, ClassificationSavings,
		ClassificationIncome, ClassificationDebt, ClassificationOther,
		ClassificationUnclassified,
	}
	for _, c := range valid {
		if !ValidClassification(c) {
			t.Errorf("expected %q to be valid", c)
		}
	}
	for _, c := range []Classification{"essential", "fixed", "", "Needs"} {
		if ValidClassification(c) {
			t.Errorf("expected %q to be invalid", c)
		}
	}
}