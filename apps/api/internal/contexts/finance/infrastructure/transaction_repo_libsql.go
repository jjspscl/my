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
	_, err := executor(ctx, r.db).ExecContext(ctx,
		`INSERT INTO transactions (id, user_email, amount_cents, currency, category, description, type, wallet_id, transaction_date, created_at, idempotency_key)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tx.ID, tx.UserEmail, tx.AmountCents, tx.Currency, tx.Category,
		tx.Description, tx.Type, tx.WalletID, tx.TransactionDate.Format("2006-01-02"), tx.CreatedAt,
		nullableString(optionalString(tx.IdempotencyKey)),
	)
	if err != nil {
		return fmt.Errorf("save transaction: %w", err)
	}
	return nil
}

func (r *TransactionRepoLibSQL) FindByID(ctx context.Context, id string) (*domain.Transaction, error) {
	row := executor(ctx, r.db).QueryRowContext(ctx,
		`SELECT t.id, t.user_email, t.amount_cents, t.currency, t.category, t.description, t.type, t.wallet_id, COALESCE(w.name, '') as wallet_name, t.transaction_date, t.created_at
		 FROM transactions t
		 LEFT JOIN wallets w ON t.wallet_id = w.id
		 WHERE t.id = ?`, id,
	)

	return scanTransaction(row)
}

func (r *TransactionRepoLibSQL) FindByIdempotencyKey(ctx context.Context, userEmail, key string) (*domain.Transaction, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT t.id, t.user_email, t.amount_cents, t.currency, t.category, t.description, t.type, t.wallet_id, COALESCE(w.name, '') as wallet_name, t.transaction_date, t.created_at
		 FROM transactions t
		 LEFT JOIN wallets w ON t.wallet_id = w.id
		 WHERE t.user_email = ? AND t.idempotency_key = ?`, userEmail, key,
	)

	tx, err := scanTransaction(row)
	if err != nil && err.Error() == "transaction not found" {
		return nil, nil
	}
	return tx, err
}

func (r *TransactionRepoLibSQL) ListByUserAndDateRange(ctx context.Context, userEmail string, from, to time.Time, limit, offset int) ([]*domain.Transaction, error) {
	rows, err := executor(ctx, r.db).QueryContext(ctx,
		`SELECT t.id, t.user_email, t.amount_cents, t.currency, t.category, t.description, t.type, t.wallet_id, COALESCE(w.name, '') as wallet_name, t.transaction_date, t.created_at
		 FROM transactions t
		 LEFT JOIN wallets w ON t.wallet_id = w.id
		 WHERE t.user_email = ? AND t.transaction_date >= ? AND t.transaction_date <= ?
		 ORDER BY t.transaction_date DESC, t.created_at DESC
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
	result, err := executor(ctx, r.db).ExecContext(ctx,
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

// GetTodayTotals returns per-currency income/expense/net totals for a single
// date. Aggregates are grouped by currency so mixed-currency days are never
// silently summed; the service decides how to present the result.
func (r *TransactionRepoLibSQL) GetTodayTotals(ctx context.Context, userEmail string, date time.Time) ([]domain.CurrencyTotal, error) {
	dateStr := date.Format("2006-01-02")
	rows, err := executor(ctx, r.db).QueryContext(ctx,
		`SELECT
			currency,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount_cents ELSE 0 END), 0) as expense_cents,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount_cents ELSE 0 END), 0) as income_cents,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount_cents ELSE -amount_cents END), 0) as total_cents
		 FROM transactions
		 WHERE user_email = ? AND transaction_date = ?
		 GROUP BY currency
		 ORDER BY currency`,
		userEmail, dateStr,
	)
	if err != nil {
		return nil, fmt.Errorf("get today totals: %w", err)
	}
	defer rows.Close()

	var totals []domain.CurrencyTotal
	for rows.Next() {
		var t domain.CurrencyTotal
		if err := rows.Scan(&t.Currency, &t.ExpenseCents, &t.IncomeCents, &t.TotalCents); err != nil {
			return nil, fmt.Errorf("scan today total: %w", err)
		}
		totals = append(totals, t)
	}
	return totals, rows.Err()
}

type scannable interface {
	Scan(dest ...any) error
}

func scanTransaction(row scannable) (*domain.Transaction, error) {
	var t domain.Transaction
	var dateStr string
	var createdAtStr string
	var walletID *string

	if err := row.Scan(&t.ID, &t.UserEmail, &t.AmountCents, &t.Currency,
		&t.Category, &t.Description, &t.Type, &walletID, &t.WalletName, &dateStr, &createdAtStr); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transaction not found")
		}
		return nil, fmt.Errorf("scan transaction: %w", err)
	}

	if walletID != nil {
		t.WalletID = *walletID
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

// nullableString converts *string to interface{} for SQL NULL handling.
func nullableString(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
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
