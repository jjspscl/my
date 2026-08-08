package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
)

// AnalyticsRepoLibSQL is the read-model implementation for the analytics core.
// All aggregates GROUP BY currency and use half-open [from, to) ranges so the
// (user_email, transaction_date) index applies.
type AnalyticsRepoLibSQL struct {
	db *sql.DB
}

func NewAnalyticsRepoLibSQL(db *sql.DB) *AnalyticsRepoLibSQL {
	return &AnalyticsRepoLibSQL{db: db}
}

// GetCashFlow returns per-currency income/expense/net over [from, to).
func (r *AnalyticsRepoLibSQL) GetCashFlow(ctx context.Context, userEmail string, from, to time.Time) ([]domain.CurrencyTotal, error) {
	rows, err := executor(ctx, r.db).QueryContext(ctx, `
		SELECT
			currency,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount_cents ELSE 0 END), 0) as expense_cents,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount_cents ELSE 0 END), 0) as income_cents,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount_cents ELSE -amount_cents END), 0) as total_cents
		FROM transactions
		WHERE user_email = ? AND transaction_date >= ? AND transaction_date < ?
		GROUP BY currency
		ORDER BY currency`,
		userEmail, from.Format("2006-01-02"), to.Format("2006-01-02"),
	)
	if err != nil {
		return nil, fmt.Errorf("get cash flow: %w", err)
	}
	defer rows.Close()

	var totals []domain.CurrencyTotal
	for rows.Next() {
		var t domain.CurrencyTotal
		if err := rows.Scan(&t.Currency, &t.ExpenseCents, &t.IncomeCents, &t.TotalCents); err != nil {
			return nil, fmt.Errorf("scan cash flow: %w", err)
		}
		totals = append(totals, t)
	}
	return totals, rows.Err()
}

// GetMonthlyCashFlow returns per-currency per-month income/expense/net over
// [from, to), ordered by month then currency.
func (r *AnalyticsRepoLibSQL) GetMonthlyCashFlow(ctx context.Context, userEmail string, from, to time.Time) ([]domain.MonthlyCashFlow, error) {
	// strftime is applied only to rows already narrowed by the indexed range
	// predicate, so it does not defeat the index.
	rows, err := executor(ctx, r.db).QueryContext(ctx, `
		SELECT
			strftime('%Y-%m', transaction_date) as month,
			currency,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount_cents ELSE 0 END), 0) as expense_cents,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount_cents ELSE 0 END), 0) as income_cents,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount_cents ELSE -amount_cents END), 0) as total_cents
		FROM transactions
		WHERE user_email = ? AND transaction_date >= ? AND transaction_date < ?
		GROUP BY month, currency
		ORDER BY month, currency`,
		userEmail, from.Format("2006-01-02"), to.Format("2006-01-02"),
	)
	if err != nil {
		return nil, fmt.Errorf("get monthly cash flow: %w", err)
	}
	defer rows.Close()

	var flows []domain.MonthlyCashFlow
	for rows.Next() {
		var f domain.MonthlyCashFlow
		if err := rows.Scan(&f.Month, &f.Currency, &f.ExpenseCents, &f.IncomeCents, &f.NetCents); err != nil {
			return nil, fmt.Errorf("scan monthly cash flow: %w", err)
		}
		flows = append(flows, f)
	}
	return flows, rows.Err()
}

// GetSpendingByClassification returns expense cents per currency and
// classification over [from, to). A category with no finance_categories row
// joins as NULL and is reported as ClassificationUnclassified.
func (r *AnalyticsRepoLibSQL) GetSpendingByClassification(ctx context.Context, userEmail string, from, to time.Time) ([]domain.ClassificationSpend, error) {
	rows, err := executor(ctx, r.db).QueryContext(ctx, `
		SELECT
			t.currency,
			COALESCE(c.classification, 'unclassified') as classification,
			COALESCE(SUM(t.amount_cents), 0) as amount_cents
		FROM transactions t
		LEFT JOIN finance_categories c ON c.name = t.category
		WHERE t.user_email = ? AND t.type = 'expense' AND t.transaction_date >= ? AND t.transaction_date < ?
		GROUP BY t.currency, classification
		ORDER BY t.currency, classification`,
		userEmail, from.Format("2006-01-02"), to.Format("2006-01-02"),
	)
	if err != nil {
		return nil, fmt.Errorf("get spending by classification: %w", err)
	}
	defer rows.Close()

	var spends []domain.ClassificationSpend
	for rows.Next() {
		var s domain.ClassificationSpend
		if err := rows.Scan(&s.Currency, &s.Classification, &s.AmountCents); err != nil {
			return nil, fmt.Errorf("scan classification spend: %w", err)
		}
		spends = append(spends, s)
	}
	return spends, rows.Err()
}

// GetUnclassifiedSpending returns the unclassified-versus-total expense split
// per currency over [from, to).
func (r *AnalyticsRepoLibSQL) GetUnclassifiedSpending(ctx context.Context, userEmail string, from, to time.Time) ([]domain.UnclassifiedSpending, error) {
	rows, err := executor(ctx, r.db).QueryContext(ctx, `
		SELECT
			t.currency,
			COALESCE(SUM(CASE WHEN c.classification IS NULL OR c.classification = 'unclassified' THEN t.amount_cents ELSE 0 END), 0) as unclassified_cents,
			COALESCE(SUM(t.amount_cents), 0) as total_cents
		FROM transactions t
		LEFT JOIN finance_categories c ON c.name = t.category
		WHERE t.user_email = ? AND t.type = 'expense' AND t.transaction_date >= ? AND t.transaction_date < ?
		GROUP BY t.currency
		ORDER BY t.currency`,
		userEmail, from.Format("2006-01-02"), to.Format("2006-01-02"),
	)
	if err != nil {
		return nil, fmt.Errorf("get unclassified spending: %w", err)
	}
	defer rows.Close()

	var splits []domain.UnclassifiedSpending
	for rows.Next() {
		var u domain.UnclassifiedSpending
		if err := rows.Scan(&u.Currency, &u.UnclassifiedCents, &u.TotalCents); err != nil {
			return nil, fmt.Errorf("scan unclassified spending: %w", err)
		}
		splits = append(splits, u)
	}
	return splits, rows.Err()
}

// GetTopUnclassifiedCategories returns the largest unclassified expense
// categories over [from, to), ordered by amount descending.
func (r *AnalyticsRepoLibSQL) GetTopUnclassifiedCategories(ctx context.Context, userEmail string, from, to time.Time, limit int) ([]domain.CategorySpend, error) {
	rows, err := executor(ctx, r.db).QueryContext(ctx, `
		SELECT t.category, COALESCE(SUM(t.amount_cents), 0) as amount_cents
		FROM transactions t
		LEFT JOIN finance_categories c ON c.name = t.category
		WHERE t.user_email = ? AND t.type = 'expense' AND t.transaction_date >= ? AND t.transaction_date < ?
		  AND (c.classification IS NULL OR c.classification = 'unclassified')
		GROUP BY t.category
		ORDER BY amount_cents DESC
		LIMIT ?`,
		userEmail, from.Format("2006-01-02"), to.Format("2006-01-02"), limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get top unclassified categories: %w", err)
	}
	defer rows.Close()

	var cats []domain.CategorySpend
	for rows.Next() {
		var c domain.CategorySpend
		if err := rows.Scan(&c.Category, &c.AmountCents); err != nil {
			return nil, fmt.Errorf("scan top unclassified category: %w", err)
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

// GetCategoryMonthlySpend returns monthly expense for one category in one
// currency over [from, to), ordered by month.
func (r *AnalyticsRepoLibSQL) GetCategoryMonthlySpend(ctx context.Context, userEmail, category, currency string, from, to time.Time) ([]domain.MonthlyAmount, error) {
	rows, err := executor(ctx, r.db).QueryContext(ctx, `
		SELECT strftime('%Y-%m', transaction_date) as month, COALESCE(SUM(amount_cents), 0) as amount_cents
		FROM transactions
		WHERE user_email = ? AND type = 'expense' AND currency = ? AND category = ?
		  AND transaction_date >= ? AND transaction_date < ?
		GROUP BY month
		ORDER BY month`,
		userEmail, currency, category, from.Format("2006-01-02"), to.Format("2006-01-02"),
	)
	if err != nil {
		return nil, fmt.Errorf("get category monthly spend: %w", err)
	}
	defer rows.Close()

	var amounts []domain.MonthlyAmount
	for rows.Next() {
		var a domain.MonthlyAmount
		if err := rows.Scan(&a.Month, &a.AmountCents); err != nil {
			return nil, fmt.Errorf("scan category monthly spend: %w", err)
		}
		amounts = append(amounts, a)
	}
	return amounts, rows.Err()
}

// GetUnbudgetedSpend returns expense in categories that have no budget
// allocation for the given month, restricted to one currency.
func (r *AnalyticsRepoLibSQL) GetUnbudgetedSpend(ctx context.Context, userEmail, currency, month string, from, to time.Time) (int64, error) {
	var total int64
	err := executor(ctx, r.db).QueryRowContext(ctx, `
		SELECT COALESCE(SUM(t.amount_cents), 0)
		FROM transactions t
		WHERE t.user_email = ? AND t.type = 'expense' AND t.currency = ?
		  AND t.transaction_date >= ? AND t.transaction_date < ?
		  AND t.category NOT IN (
			SELECT bc.category
			FROM budget_categories bc
			JOIN budgets b ON b.id = bc.budget_id
			WHERE b.user_email = ? AND b.month = ?
		  )`,
		userEmail, currency, from.Format("2006-01-02"), to.Format("2006-01-02"), userEmail, month,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("get unbudgeted spend: %w", err)
	}
	return total, nil
}
