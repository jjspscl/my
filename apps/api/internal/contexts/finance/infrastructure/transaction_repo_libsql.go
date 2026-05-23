package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
)

type TransactionRepoLibSQL struct {
	db *sql.DB
}

func NewTransactionRepoLibSQL(db *sql.DB) *TransactionRepoLibSQL {
	return &TransactionRepoLibSQL{db: db}
}

func (r *TransactionRepoLibSQL) Save(ctx context.Context, tx *domain.Transaction) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO transactions (id, user_email, amount_cents, currency, category, description, type, transaction_date, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tx.ID, tx.UserEmail, tx.AmountCents, tx.Currency, tx.Category,
		tx.Description, tx.Type, tx.TransactionDate.Format("2006-01-02"), tx.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("save transaction: %w", err)
	}
	return nil
}

func (r *TransactionRepoLibSQL) FindByID(ctx context.Context, id string) (*domain.Transaction, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_email, amount_cents, currency, category, description, type, transaction_date, created_at
		 FROM transactions WHERE id = ?`, id,
	)

	return scanTransaction(row)
}

func (r *TransactionRepoLibSQL) ListByUserAndDateRange(ctx context.Context, userEmail string, from, to time.Time, limit, offset int) ([]*domain.Transaction, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_email, amount_cents, currency, category, description, type, transaction_date, created_at
		 FROM transactions
		 WHERE user_email = ? AND transaction_date >= ? AND transaction_date <= ?
		 ORDER BY transaction_date DESC, created_at DESC
		 LIMIT ? OFFSET ?`,
		userEmail, from.Format("2006-01-02"), to.Format("2006-01-02"), limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	var txs []*domain.Transaction
	for rows.Next() {
		tx, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}

	return txs, rows.Err()
}

func (r *TransactionRepoLibSQL) Delete(ctx context.Context, id, userEmail string) error {
	result, err := r.db.ExecContext(ctx,
		"DELETE FROM transactions WHERE id = ? AND user_email = ?", id, userEmail,
	)
	if err != nil {
		return fmt.Errorf("delete transaction: %w", err)
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("transaction not found")
	}

	return nil
}

func (r *TransactionRepoLibSQL) GetTodayTotal(ctx context.Context, userEmail string, date time.Time) (*domain.DailyTotal, error) {
	dateStr := date.Format("2006-01-02")
	row := r.db.QueryRowContext(ctx,
		`SELECT
			? as date,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount_cents ELSE 0 END), 0) as expense_cents,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount_cents ELSE 0 END), 0) as income_cents,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount_cents ELSE -amount_cents END), 0) as total_cents
		 FROM transactions
		 WHERE user_email = ? AND transaction_date = ?`,
		dateStr, userEmail, dateStr,
	)

	var total domain.DailyTotal
	if err := row.Scan(&total.Date, &total.ExpenseCents, &total.IncomeCents, &total.TotalCents); err != nil {
		return nil, fmt.Errorf("get today total: %w", err)
	}

	// default currency from a recent transaction
	currencyRow := r.db.QueryRowContext(ctx,
		"SELECT currency FROM transactions WHERE user_email = ? AND transaction_date = ? LIMIT 1",
		userEmail, dateStr,
	)
	var currency string
	if err := currencyRow.Scan(&currency); err == nil {
		total.Currency = currency
	}

	return &total, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanTransaction(row scannable) (*domain.Transaction, error) {
	var t domain.Transaction
	var dateStr string
	var createdAtStr string

	if err := row.Scan(&t.ID, &t.UserEmail, &t.AmountCents, &t.Currency,
		&t.Category, &t.Description, &t.Type, &dateStr, &createdAtStr); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transaction not found")
		}
		return nil, fmt.Errorf("scan transaction: %w", err)
	}

	parsed, err := parseDatetime(dateStr)
	if err != nil {
		return nil, fmt.Errorf("parse date: %w", err)
	}
	t.TransactionDate = parsed

	createdAt, err := parseDatetime(createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	t.CreatedAt = createdAt

	return &t, nil
}

// parseDatetime tries common SQLite datetime formats.
func parseDatetime(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized datetime: %s", s)
}
