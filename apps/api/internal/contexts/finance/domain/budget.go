package domain

import (
	"fmt"
	"regexp"
	"time"
)

var monthRegex = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

type Budget struct {
	ID        string
	UserEmail string
	Month     string // YYYY-MM
	Currency  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type BudgetCategory struct {
	ID              string
	BudgetID        string
	Category        string
	AllocatedCents  int64
	RolloverEnabled bool
}

type BudgetCategorySummary struct {
	Category        string
	AllocatedCents  int64
	SpentCents      int64
	RemainingCents  int64
	RolloverEnabled bool
}

type BudgetSummary struct {
	Month               string
	TotalAllocatedCents int64
	TotalSpentCents     int64
	TotalRemainingCents int64
	Categories          []BudgetCategorySummary
}

// NewBudget validates and creates a Budget.
func NewBudget(id, userEmail, month, currency string) (*Budget, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if userEmail == "" {
		return nil, fmt.Errorf("user email is required")
	}
	if !monthRegex.MatchString(month) {
		return nil, fmt.Errorf("invalid month format, expected YYYY-MM: %s", month)
	}
	if currency == "" {
		currency = "PHP"
	}

	return &Budget{
		ID:        id,
		UserEmail: userEmail,
		Month:     month,
		Currency:  currency,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}, nil
}

// NewBudgetCategory validates and creates a BudgetCategory.
func NewBudgetCategory(id, budgetID, category string, allocatedCents int64, rolloverEnabled bool) (*BudgetCategory, error) {
	if category == "" {
		return nil, fmt.Errorf("category is required")
	}
	if len(category) > MaxCategoryLen {
		return nil, fmt.Errorf("category too long (max %d)", MaxCategoryLen)
	}
	if allocatedCents < 0 {
		return nil, fmt.Errorf("allocated amount cannot be negative")
	}

	return &BudgetCategory{
		ID:              id,
		BudgetID:        budgetID,
		Category:        category,
		AllocatedCents:  allocatedCents,
		RolloverEnabled: rolloverEnabled,
	}, nil
}
