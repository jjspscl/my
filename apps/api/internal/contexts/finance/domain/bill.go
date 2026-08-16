package domain

import (
	"fmt"
	"strings"
	"time"
)

type Frequency string

const (
	FrequencyMonthly Frequency = "monthly"
	FrequencyWeekly  Frequency = "weekly"
	FrequencyYearly  Frequency = "yearly"

	MaxBillNameLen     = 200
	MaxMatchPatternLen = 500

	DayOfMonthMin = 1
	DayOfMonthMax = 31
)

type OccurrenceStatus string

const (
	OccurrencePending OccurrenceStatus = "pending"
	OccurrenceOverdue OccurrenceStatus = "overdue"
	OccurrencePaid    OccurrenceStatus = "paid"
	OccurrenceSkipped OccurrenceStatus = "skipped"
)

// Payment link sources — how a bill payment came to reference a transaction.
const (
	// PaymentLinkAuto means the link was inferred by bill auto-match; editing
	// or deleting the transaction may safely remove the payment record.
	PaymentLinkAuto = "auto"
	// PaymentLinkManual means the user explicitly linked the payment to the
	// transaction; deleting the transaction keeps the paid occurrence
	// (transaction_id becomes NULL via FK).
	PaymentLinkManual = "manual"
	// PaymentLinkLegacy covers rows created before link sources existed.
	PaymentLinkLegacy = "legacy"
)

type RecurringBill struct {
	ID           string
	UserEmail    string
	Name         string
	Category     string
	AmountCents  int64
	Currency     string
	Frequency    Frequency
	DayOfMonth   int
	StartDate    time.Time
	EndDate      *time.Time
	AutoMatch    bool
	MatchPattern *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type BillPayment struct {
	ID                    string
	BillID                string
	TransactionID         *string
	TransactionLinkSource string
	DueDate               time.Time
	PaidDate              *time.Time
	AmountCents           int64
	Status                OccurrenceStatus
	CreatedAt             time.Time
}

type BillWithPayment struct {
	Bill    RecurringBill
	Payment *BillPayment // nil if no payment record exists for this due date
}

type UpcomingBill struct {
	Bill            RecurringBill
	DueDate         time.Time
	Status          OccurrenceStatus
	PaidAmountCents *int64
	PaidDate        *time.Time
}

// NewRecurringBill validates and creates a RecurringBill.
func NewRecurringBill(id, userEmail, name, category string, amountCents int64, currency string, freq Frequency, dayOfMonth int, startDate time.Time, endDate *time.Time, autoMatch bool, matchPattern *string) (*RecurringBill, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if userEmail == "" {
		return nil, fmt.Errorf("user email is required")
	}
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	if len(name) > MaxBillNameLen {
		return nil, fmt.Errorf("name too long (max %d)", MaxBillNameLen)
	}
	if strings.TrimSpace(category) == "" {
		return nil, fmt.Errorf("category is required")
	}
	if len(category) > MaxCategoryLen {
		return nil, fmt.Errorf("category too long (max %d)", MaxCategoryLen)
	}
	if amountCents <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}
	if currency == "" {
		currency = "PHP"
	}
	if freq != FrequencyMonthly && freq != FrequencyWeekly && freq != FrequencyYearly {
		return nil, fmt.Errorf("invalid frequency: %s", freq)
	}
	if freq != FrequencyWeekly {
		if dayOfMonth < DayOfMonthMin || dayOfMonth > DayOfMonthMax {
			return nil, fmt.Errorf("day of month out of range (1-31): %d", dayOfMonth)
		}
	}
	if endDate != nil && !endDate.After(startDate) {
		return nil, fmt.Errorf("end date must be after start date")
	}
	if matchPattern != nil && len(*matchPattern) > MaxMatchPatternLen {
		return nil, fmt.Errorf("match pattern too long (max %d)", MaxMatchPatternLen)
	}

	return &RecurringBill{
		ID:           id,
		UserEmail:    userEmail,
		Name:         strings.TrimSpace(name),
		Category:     strings.TrimSpace(category),
		AmountCents:  amountCents,
		Currency:     currency,
		Frequency:    freq,
		DayOfMonth:   dayOfMonth,
		StartDate:    startDate,
		EndDate:      endDate,
		AutoMatch:    autoMatch,
		MatchPattern: matchPattern,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}, nil
}

// NewBillPayment validates and creates a BillPayment.
func NewBillPayment(id, billID string, dueDate time.Time, amountCents int64) (*BillPayment, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if billID == "" {
		return nil, fmt.Errorf("bill id is required")
	}
	if dueDate.IsZero() {
		return nil, fmt.Errorf("due date is required")
	}
	if amountCents <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	return &BillPayment{
		ID:          id,
		BillID:      billID,
		DueDate:     dueDate,
		AmountCents: amountCents,
		Status:      OccurrencePending,
		CreatedAt:   time.Now().UTC(),
	}, nil
}
