package application

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
	"github.com/jjspscl/my/internal/platform/timeutil"
)

type BillService struct {
	billRepo domain.BillRepository
	currency string
	clock    *timeutil.Clock
}

func NewBillService(billRepo domain.BillRepository) *BillService {
	return &BillService{billRepo: billRepo, currency: "PHP", clock: timeutil.New(time.UTC)}
}

// WithCurrency sets the default currency for new bills. Bills are expectations
// without a wallet, so they default to the reporting base currency.
func (s *BillService) WithCurrency(c string) *BillService {
	if c != "" {
		s.currency = c
	}
	return s
}

// WithClock pins the calendar used for due-date computation.
func (s *BillService) WithClock(c *timeutil.Clock) *BillService {
	s.clock = c
	return s
}

type CreateBillInput struct {
	Name         string
	Category     string
	AmountCents  int64
	Currency     string
	Frequency    domain.Frequency
	DayOfMonth   int
	StartDate    time.Time
	EndDate      *time.Time
	AutoMatch    bool
	MatchPattern *string
}

type UpdateBillInput struct {
	ID           string
	Name         string
	Category     string
	AmountCents  int64
	Currency     string
	Frequency    domain.Frequency
	DayOfMonth   int
	StartDate    time.Time
	EndDate      *time.Time
	AutoMatch    bool
	MatchPattern *string
}

func (s *BillService) Create(ctx context.Context, userEmail string, input CreateBillInput) (*domain.RecurringBill, error) {
	currency := input.Currency
	if currency == "" {
		currency = s.currency
	}

	bill, err := domain.NewRecurringBill(
		uuid.New().String(),
		userEmail,
		input.Name,
		input.Category,
		input.AmountCents,
		currency,
		input.Frequency,
		input.DayOfMonth,
		input.StartDate,
		input.EndDate,
		input.AutoMatch,
		input.MatchPattern,
	)
	if err != nil {
		return nil, err
	}

	if err := s.billRepo.SaveBill(ctx, bill); err != nil {
		return nil, fmt.Errorf("save bill: %w", err)
	}

	return bill, nil
}

func (s *BillService) Update(ctx context.Context, userEmail string, input UpdateBillInput) (*domain.RecurringBill, error) {
	existing, err := s.billRepo.FindBillByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("find bill: %w", err)
	}
	if existing.UserEmail != userEmail {
		return nil, fmt.Errorf("bill not found")
	}

	currency := input.Currency
	if currency == "" {
		currency = existing.Currency
	}

	bill, err := domain.NewRecurringBill(
		input.ID,
		userEmail,
		input.Name,
		input.Category,
		input.AmountCents,
		currency,
		input.Frequency,
		input.DayOfMonth,
		input.StartDate,
		input.EndDate,
		input.AutoMatch,
		input.MatchPattern,
	)
	if err != nil {
		return nil, err
	}
	bill.CreatedAt = existing.CreatedAt

	if err := s.billRepo.UpdateBill(ctx, bill); err != nil {
		return nil, fmt.Errorf("update bill: %w", err)
	}

	return bill, nil
}

func (s *BillService) Delete(ctx context.Context, id, userEmail string) error {
	return s.billRepo.DeleteBill(ctx, id, userEmail)
}

func (s *BillService) List(ctx context.Context, userEmail string) ([]*domain.RecurringBill, error) {
	return s.billRepo.ListBills(ctx, userEmail)
}

type UpcomingBillResult struct {
	Bill            domain.RecurringBill
	DueDate         time.Time
	Status          domain.OccurrenceStatus
	PaidAmountCents *int64
	PaidDate        *time.Time
}

// GetUpcoming returns upcoming bills with computed due dates and status.
// Occurrences are generated per bill, then all payment records for the window
// are fetched in one batched query (no per-occurrence FindPayment N+1).
func (s *BillService) GetUpcoming(ctx context.Context, userEmail string, daysAhead int) ([]UpcomingBillResult, error) {
	bills, err := s.billRepo.ListBills(ctx, userEmail)
	if err != nil {
		return nil, fmt.Errorf("list bills: %w", err)
	}

	now := s.clock.TodayStart()
	cutoff := now.AddDate(0, 0, daysAhead)
	windowStart := now.AddDate(0, 0, -60)

	// Collect every due date we will need to look up, per bill.
	type occurrence struct {
		bill    *domain.RecurringBill
		dueDate time.Time
	}
	var occurrences []occurrence
	billIDs := make([]string, 0, len(bills))
	seen := make(map[string]bool, len(bills))
	for _, bill := range bills {
		if !seen[bill.ID] {
			seen[bill.ID] = true
			billIDs = append(billIDs, bill.ID)
		}
		for _, dueDate := range generateOccurrences(bill, windowStart, cutoff) {
			if bill.EndDate != nil && dueDate.After(*bill.EndDate) {
				continue
			}
			occurrences = append(occurrences, occurrence{bill: bill, dueDate: dueDate})
		}
	}

	// One batched query for all payments in the window.
	payments, err := s.billRepo.ListPaymentsByBills(ctx, billIDs, windowStart, cutoff)
	if err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}
	paymentByKey := make(map[string]*domain.BillPayment, len(payments))
	for _, p := range payments {
		paymentByKey[p.BillID+"|"+p.DueDate.Format("2006-01-02")] = p
	}

	results := make([]UpcomingBillResult, 0, len(occurrences))
	for _, occ := range occurrences {
		dueStr := occ.dueDate.Format("2006-01-02")
		payment := paymentByKey[occ.bill.ID+"|"+dueStr]

		status := domain.OccurrencePending
		var paidAmount *int64
		var paidDate *time.Time

		if payment != nil {
			switch payment.Status {
			case domain.OccurrencePaid:
				status = domain.OccurrencePaid
				paidAmount = &payment.AmountCents
				paidDate = payment.PaidDate
			case domain.OccurrenceOverdue:
				status = domain.OccurrenceOverdue
			case domain.OccurrenceSkipped:
				status = domain.OccurrenceSkipped
			default:
				if occ.dueDate.Before(now) {
					status = domain.OccurrenceOverdue
				}
			}
		} else if occ.dueDate.Before(now) {
			status = domain.OccurrenceOverdue
		}

		results = append(results, UpcomingBillResult{
			Bill:            *occ.bill,
			DueDate:         occ.dueDate,
			Status:          status,
			PaidAmountCents: paidAmount,
			PaidDate:        paidDate,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].DueDate.Before(results[j].DueDate)
	})

	return results, nil
}

func (s *BillService) MarkPaid(ctx context.Context, billID, userEmail string, dueDate time.Time, transactionID *string) (*domain.BillPayment, error) {
	bill, err := s.billRepo.FindBillByID(ctx, billID)
	if err != nil {
		return nil, fmt.Errorf("find bill: %w", err)
	}
	if bill.UserEmail != userEmail {
		return nil, fmt.Errorf("bill not found")
	}

	dueStr := dueDate.Format("2006-01-02")

	existing, err := s.billRepo.FindPayment(ctx, billID, dueStr)
	if err != nil && err.Error() != "payment not found" {
		return nil, fmt.Errorf("find payment: %w", err)
	}

	now := s.clock.Now()

	payment := existing
	if payment == nil {
		payment, err = domain.NewBillPayment(uuid.New().String(), billID, dueDate, bill.AmountCents)
		if err != nil {
			return nil, err
		}
	}

	payment.Status = domain.OccurrencePaid
	payment.PaidDate = &now
	payment.TransactionID = transactionID

	if err := s.billRepo.SavePayment(ctx, payment); err != nil {
		return nil, fmt.Errorf("save payment: %w", err)
	}

	return payment, nil
}

// TryAutoMatch checks if a transaction matches any bill's auto-match criteria.
func (s *BillService) TryAutoMatch(ctx context.Context, tx *domain.Transaction) {
	if tx.Type != domain.TransactionExpense {
		return
	}

	bills, err := s.billRepo.ListBills(ctx, tx.UserEmail)
	if err != nil {
		return
	}

	txDate := tx.TransactionDate.Format("2006-01-02")

	for _, bill := range bills {
		if !bill.AutoMatch {
			continue
		}

		pattern := ""
		if bill.MatchPattern != nil {
			pattern = *bill.MatchPattern
		}

		matched, err := s.billRepo.FindTransactionByMatch(ctx, tx.UserEmail, bill.Category, bill.AmountCents, txDate, pattern)
		if err != nil {
			continue
		}

		if matched != nil && matched.ID == tx.ID {
			// Calculate due date for this bill near the transaction date
			dueDate := findNearestDueDate(bill, tx.TransactionDate)
			if dueDate == nil {
				return
			}

			dueStr := dueDate.Format("2006-01-02")

			existing, err := s.billRepo.FindPayment(ctx, bill.ID, dueStr)
			if err == nil && existing != nil {
				// already paid
				return
			}

			payment, err := domain.NewBillPayment(uuid.New().String(), bill.ID, *dueDate, bill.AmountCents)
			if err != nil {
				return
			}
			payment.Status = domain.OccurrencePaid
			payment.PaidDate = &tx.TransactionDate
			payment.TransactionID = &tx.ID

			_ = s.billRepo.SavePayment(ctx, payment)
			return // only match first qualifying bill
		}
	}
}

// generateOccurrences creates due dates for a bill in the given range.
func generateOccurrences(bill *domain.RecurringBill, from, to time.Time) []time.Time {
	var dates []time.Time

	start := bill.StartDate
	if from.After(start) {
		start = from
	}

	if to.Before(start) {
		return nil
	}

	switch bill.Frequency {
	case domain.FrequencyMonthly:
		current := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(to.Year(), to.Month(), 1, 0, 0, 0, 0, time.UTC)
		for !current.After(end) {
			due := dayInMonth(current, bill.DayOfMonth)
			if (due.Equal(start) || due.After(start)) && (due.Equal(to) || !due.After(to)) {
				dates = append(dates, due)
			}
			current = current.AddDate(0, 1, 0)
		}

	case domain.FrequencyYearly:
		current := time.Date(start.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
		for !current.After(end) {
			due := dayInMonth(current, bill.DayOfMonth)
			if (due.Equal(start) || due.After(start)) && (due.Equal(to) || !due.After(to)) {
				dates = append(dates, due)
			}
			current = current.AddDate(1, 0, 0)
		}

	case domain.FrequencyWeekly:
		current := start
		for !current.After(to) {
			if (current.Equal(from) || !current.Before(from)) && (current.Equal(to) || !current.After(to)) {
				dates = append(dates, current)
			}
			current = current.AddDate(0, 0, 7)
		}
	}

	return dates
}

func dayInMonth(month time.Time, day int) time.Time {
	firstOfMonth := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstOfMonth.AddDate(0, 1, -1).Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(month.Year(), month.Month(), day, 0, 0, 0, 0, time.UTC)
}

func findNearestDueDate(bill *domain.RecurringBill, txDate time.Time) *time.Time {
	occurrences := generateOccurrences(bill, txDate.AddDate(0, 0, -15), txDate.AddDate(0, 0, 15))
	if len(occurrences) == 0 {
		return nil
	}

	// Find closest due date to txDate
	best := occurrences[0]
	bestDiff := absDuration(best.Sub(txDate))
	for _, d := range occurrences[1:] {
		diff := absDuration(d.Sub(txDate))
		if diff < bestDiff {
			best = d
			bestDiff = diff
		}
	}
	return &best
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
