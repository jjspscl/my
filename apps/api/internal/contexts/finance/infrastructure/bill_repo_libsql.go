package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
)

type BillRepoLibSQL struct {
	db *sql.DB
}

func NewBillRepoLibSQL(db *sql.DB) *BillRepoLibSQL {
	return &BillRepoLibSQL{db: db}
}

func (r *BillRepoLibSQL) SaveBill(ctx context.Context, bill *domain.RecurringBill) error {
	endDate := (*string)(nil)
	if bill.EndDate != nil {
		s := bill.EndDate.Format("2006-01-02")
		endDate = &s
	}
	matchPattern := bill.MatchPattern
	autoMatch := 0
	if bill.AutoMatch {
		autoMatch = 1
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO recurring_bills (id, user_email, name, category, amount_cents, frequency, day_of_month, start_date, end_date, auto_match, match_pattern, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, bill.ID, bill.UserEmail, bill.Name, bill.Category, bill.AmountCents, string(bill.Frequency), bill.DayOfMonth, bill.StartDate.Format("2006-01-02"), endDate, autoMatch, matchPattern, bill.CreatedAt.Format(time.RFC3339), bill.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("save bill: %w", err)
	}
	return nil
}

func (r *BillRepoLibSQL) UpdateBill(ctx context.Context, bill *domain.RecurringBill) error {
	endDate := (*string)(nil)
	if bill.EndDate != nil {
		s := bill.EndDate.Format("2006-01-02")
		endDate = &s
	}
	matchPattern := bill.MatchPattern
	autoMatch := 0
	if bill.AutoMatch {
		autoMatch = 1
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE recurring_bills SET name = ?, category = ?, amount_cents = ?, frequency = ?, day_of_month = ?, start_date = ?, end_date = ?, auto_match = ?, match_pattern = ?, updated_at = ?
		WHERE id = ? AND user_email = ?
	`, bill.Name, bill.Category, bill.AmountCents, string(bill.Frequency), bill.DayOfMonth, bill.StartDate.Format("2006-01-02"), endDate, autoMatch, matchPattern, bill.UpdatedAt.Format(time.RFC3339), bill.ID, bill.UserEmail)
	if err != nil {
		return fmt.Errorf("update bill: %w", err)
	}
	return nil
}

func (r *BillRepoLibSQL) DeleteBill(ctx context.Context, id, userEmail string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM recurring_bills WHERE id = ? AND user_email = ?", id, userEmail)
	if err != nil {
		return fmt.Errorf("delete bill: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("bill not found")
	}
	return nil
}

func (r *BillRepoLibSQL) FindBillByID(ctx context.Context, id string) (*domain.RecurringBill, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_email, name, category, amount_cents, frequency, day_of_month, start_date, end_date, auto_match, match_pattern, created_at, updated_at
		FROM recurring_bills WHERE id = ?
	`, id)
	return scanBill(row)
}

func (r *BillRepoLibSQL) ListBills(ctx context.Context, userEmail string) ([]*domain.RecurringBill, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_email, name, category, amount_cents, frequency, day_of_month, start_date, end_date, auto_match, match_pattern, created_at, updated_at
		FROM recurring_bills WHERE user_email = ? ORDER BY name ASC
	`, userEmail)
	if err != nil {
		return nil, fmt.Errorf("list bills: %w", err)
	}
	defer rows.Close()

	var bills []*domain.RecurringBill
	for rows.Next() {
		b, err := scanBill(rows)
		if err != nil {
			return nil, err
		}
		bills = append(bills, b)
	}
	return bills, rows.Err()
}

func (r *BillRepoLibSQL) SavePayment(ctx context.Context, payment *domain.BillPayment) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO bill_payments (id, bill_id, transaction_id, due_date, paid_date, amount_cents, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(bill_id, due_date) DO UPDATE SET status = ?, paid_date = ?, transaction_id = ?
	`, payment.ID, payment.BillID, payment.TransactionID, payment.DueDate.Format("2006-01-02"), nullableTime(payment.PaidDate), payment.AmountCents, string(payment.Status), payment.CreatedAt.Format(time.RFC3339), string(payment.Status), nullableTime(payment.PaidDate), payment.TransactionID)
	if err != nil {
		return fmt.Errorf("save payment: %w", err)
	}
	return nil
}

func (r *BillRepoLibSQL) FindPayment(ctx context.Context, billID, dueDate string) (*domain.BillPayment, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, bill_id, transaction_id, due_date, paid_date, amount_cents, status, created_at
		FROM bill_payments WHERE bill_id = ? AND due_date = ?
	`, billID, dueDate)
	return scanPayment(row)
}

func (r *BillRepoLibSQL) ListPaymentsByBill(ctx context.Context, billID string) ([]*domain.BillPayment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, bill_id, transaction_id, due_date, paid_date, amount_cents, status, created_at
		FROM bill_payments WHERE bill_id = ? ORDER BY due_date DESC
	`, billID)
	if err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}
	defer rows.Close()

	var payments []*domain.BillPayment
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}
	return payments, rows.Err()
}

// ListUpcomingBills returns bills with their most recent payment info (if any).
// It joins recurring_bills with bill_payments to find upcoming/overdue occurrences.
// limit controls how many bills to return.
func (r *BillRepoLibSQL) ListUpcomingBills(ctx context.Context, userEmail string, limit int) ([]*domain.BillWithPayment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			b.id, b.user_email, b.name, b.category, b.amount_cents, b.frequency, b.day_of_month,
			b.start_date, b.end_date, b.auto_match, b.match_pattern, b.created_at, b.updated_at,
			p.id, p.bill_id, p.transaction_id, p.due_date, p.paid_date, p.amount_cents, p.status, p.created_at
		FROM recurring_bills b
		LEFT JOIN (
			SELECT bill_id, id, transaction_id, due_date, paid_date, amount_cents, status, created_at
			FROM bill_payments
			WHERE (bill_id, due_date) IN (
				SELECT bill_id, MAX(due_date) FROM bill_payments GROUP BY bill_id
			)
		) p ON p.bill_id = b.id
		WHERE b.user_email = ?
		ORDER BY COALESCE(p.due_date, b.start_date) ASC
		LIMIT ?
	`, userEmail, limit)
	if err != nil {
		return nil, fmt.Errorf("list upcoming bills: %w", err)
	}
	defer rows.Close()

	var results []*domain.BillWithPayment
	for rows.Next() {
		var b domain.RecurringBill
		var p domain.BillPayment
		var startDate, endDate, createdAt, updatedAt string
		var payID, payBillID, payDueDate, payCreatedAt sql.NullString
		var payTransactionID, payPaidDate, payStatus sql.NullString
		var payAmountCents sql.NullInt64
		autoMatch := 0
		matchPattern := (*string)(nil)

		err := rows.Scan(
			&b.ID, &b.UserEmail, &b.Name, &b.Category, &b.AmountCents, (*string)(&b.Frequency),
			&b.DayOfMonth, &startDate, &endDate, &autoMatch, &matchPattern,
			&createdAt, &updatedAt,
			&payID, &payBillID, &payTransactionID, &payDueDate, &payPaidDate,
			&payAmountCents, &payStatus, &payCreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan upcoming bill: %w", err)
		}

		b.AutoMatch = autoMatch == 1
		b.MatchPattern = matchPattern
		b.StartDate, _ = time.Parse("2006-01-02", startDate)
		if endDate != "" {
			parsed, _ := time.Parse("2006-01-02", endDate)
			b.EndDate = &parsed
		}
		b.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		b.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

		entry := &domain.BillWithPayment{Bill: b}
		if payID.Valid {
			p.ID = payID.String
			p.BillID = payBillID.String
			if payTransactionID.Valid {
				p.TransactionID = &payTransactionID.String
			}
			if payDueDate.Valid {
				p.DueDate, _ = time.Parse("2006-01-02", payDueDate.String)
			}
			if payPaidDate.Valid {
				parsed, _ := time.Parse("2006-01-02", payPaidDate.String)
				p.PaidDate = &parsed
			}
			if payAmountCents.Valid {
				p.AmountCents = payAmountCents.Int64
			}
			if payStatus.Valid {
				p.Status = domain.OccurrenceStatus(payStatus.String)
			}
			if payCreatedAt.Valid {
				p.CreatedAt, _ = time.Parse(time.RFC3339, payCreatedAt.String)
			}
			entry.Payment = &p
		}

		results = append(results, entry)
	}
	return results, rows.Err()
}

// FindTransactionByMatch looks for a transaction matching category, amount (within 10%), and optional pattern within ±5 days of date.
func (r *BillRepoLibSQL) FindTransactionByMatch(ctx context.Context, userEmail, category string, amountCents int64, date string, pattern string) (*domain.Transaction, error) {
	minAmount := amountCents * 90 / 100
	maxAmount := amountCents * 110 / 100

	var descFilter string
	var args []interface{}
	if pattern != "" {
		descFilter = " AND LOWER(description) LIKE ?"
		args = append(args, "%"+pattern+"%")
	}

	args = append(args, userEmail, category, minAmount, maxAmount, date)

	row := r.db.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT id, user_email, amount_cents, currency, category, description, type, transaction_date, created_at
		FROM transactions
		WHERE user_email = ?
		  AND type = 'expense'
		  AND category = ?
		  AND amount_cents BETWEEN ? AND ?
		  AND transaction_date BETWEEN date(?, '-5 days') AND date(?, '+5 days')
		  %s
		ORDER BY transaction_date DESC
		LIMIT 1
	`, descFilter), append(args, date)...)

	return scanTransaction(row)
}

// --- scanner helpers ---

func scanBill(row scannable) (*domain.RecurringBill, error) {
	var b domain.RecurringBill
	var startDate, endDate, createdAt, updatedAt string
	autoMatch := 0
	matchPattern := (*string)(nil)

	err := row.Scan(
		&b.ID, &b.UserEmail, &b.Name, &b.Category, &b.AmountCents, (*string)(&b.Frequency),
		&b.DayOfMonth, &startDate, &endDate, &autoMatch, &matchPattern,
		&createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("bill not found")
	}
	if err != nil {
		return nil, fmt.Errorf("scan bill: %w", err)
	}

	b.AutoMatch = autoMatch == 1
	b.MatchPattern = matchPattern
	b.StartDate, _ = time.Parse("2006-01-02", startDate)
	if endDate != "" {
		parsed, _ := time.Parse("2006-01-02", endDate)
		b.EndDate = &parsed
	}
	b.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	b.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &b, nil
}

func scanPayment(row scannable) (*domain.BillPayment, error) {
	var p domain.BillPayment
	var dueDate, paidDate, createdAt string
	var transactionID *string

	err := row.Scan(&p.ID, &p.BillID, &transactionID, &dueDate, &paidDate, &p.AmountCents, (*string)(&p.Status), &createdAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("payment not found")
	}
	if err != nil {
		return nil, fmt.Errorf("scan payment: %w", err)
	}

	p.TransactionID = transactionID
	p.DueDate, _ = time.Parse("2006-01-02", dueDate)
	if paidDate != "" {
		parsed, _ := time.Parse("2006-01-02", paidDate)
		p.PaidDate = &parsed
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	return &p, nil
}

func nullableTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format("2006-01-02")
}
