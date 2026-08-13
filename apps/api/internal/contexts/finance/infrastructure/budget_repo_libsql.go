package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
)

type BudgetRepoLibSQL struct {
	db *sql.DB
}

func NewBudgetRepoLibSQL(db *sql.DB) *BudgetRepoLibSQL {
	return &BudgetRepoLibSQL{db: db}
}

func (r *BudgetRepoLibSQL) UpsertBudget(ctx context.Context, b *domain.Budget) error {
	_, err := executor(ctx, r.db).ExecContext(ctx, `
		INSERT INTO budgets (id, user_email, month, currency, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_email, month) DO UPDATE SET currency = ?, updated_at = ?
	`, b.ID, b.UserEmail, b.Month, b.Currency, b.CreatedAt.Format(time.RFC3339), b.UpdatedAt.Format(time.RFC3339), b.Currency, b.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("upsert budget: %w", err)
	}
	return nil
}

func (r *BudgetRepoLibSQL) FindBudgetByMonth(ctx context.Context, userEmail, month string) (*domain.Budget, error) {
	row := executor(ctx, r.db).QueryRowContext(ctx, `
		SELECT id, user_email, month, currency, created_at, updated_at
		FROM budgets WHERE user_email = ? AND month = ?
	`, userEmail, month)

	var b domain.Budget
	var createdAt, updatedAt string
	err := row.Scan(&b.ID, &b.UserEmail, &b.Month, &b.Currency, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find budget: %w", err)
	}
	b.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	b.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &b, nil
}

func (r *BudgetRepoLibSQL) UpsertBudgetCategories(ctx context.Context, budgetID string, categories []*domain.BudgetCategory) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM budget_categories WHERE budget_id = ?`, budgetID); err != nil {
		return fmt.Errorf("delete old categories: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO budget_categories (id, budget_id, category, allocated_cents, rollover_enabled)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare stmt: %w", err)
	}
	defer stmt.Close()

	for _, c := range categories {
		rollover := 0
		if c.RolloverEnabled {
			rollover = 1
		}
		if _, err := stmt.ExecContext(ctx, c.ID, budgetID, c.Category, c.AllocatedCents, rollover); err != nil {
			return fmt.Errorf("insert category %q: %w", c.Category, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (r *BudgetRepoLibSQL) GetBudgetCategories(ctx context.Context, budgetID string) ([]*domain.BudgetCategory, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, budget_id, category, allocated_cents, rollover_enabled
		FROM budget_categories WHERE budget_id = ?
	`, budgetID)
	if err != nil {
		return nil, fmt.Errorf("get categories: %w", err)
	}
	defer rows.Close()

	var result []*domain.BudgetCategory
	for rows.Next() {
		var bc domain.BudgetCategory
		var rollover int
		if err := rows.Scan(&bc.ID, &bc.BudgetID, &bc.Category, &bc.AllocatedCents, &rollover); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		bc.RolloverEnabled = rollover == 1
		result = append(result, &bc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return result, nil
}

// GetSpentByCategory sums expense transactions per category within the
// half-open [from, to) range, restricted to a single currency. The range
// predicate lets the (user_email, transaction_date) index apply, unlike a
// strftime('%Y-%m', transaction_date) filter which forces a full scan.
func (r *BudgetRepoLibSQL) GetSpentByCategory(ctx context.Context, userEmail, currency string, from, to time.Time) (map[string]int64, error) {
	rows, err := executor(ctx, r.db).QueryContext(ctx, `
		SELECT category, COALESCE(SUM(amount_cents), 0)
		FROM transactions
		WHERE user_email = ?
		  AND type = 'expense'
		  AND currency = ?
		  AND transaction_date >= ?
		  AND transaction_date < ?
		GROUP BY category
	`, userEmail, currency, from.Format("2006-01-02"), to.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("get spent by category: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var cat string
		var cents int64
		if err := rows.Scan(&cat, &cents); err != nil {
			return nil, fmt.Errorf("scan spent: %w", err)
		}
		result[cat] = cents
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return result, nil
}
