package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
)

type GoalRepoLibSQL struct {
	db *sql.DB
}

func NewGoalRepoLibSQL(db *sql.DB) *GoalRepoLibSQL {
	return &GoalRepoLibSQL{db: db}
}

func (r *GoalRepoLibSQL) SaveGoal(ctx context.Context, goal *domain.SavingsGoal) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO savings_goals (id, user_email, name, target_amount_cents, target_date, target_wallet_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, goal.ID, goal.UserEmail, goal.Name, goal.TargetAmountCents, nullableTime(goal.TargetDate), goal.TargetWalletID, goal.CreatedAt.Format(time.RFC3339), goal.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("save goal: %w", err)
	}
	return nil
}

func (r *GoalRepoLibSQL) UpdateGoal(ctx context.Context, goal *domain.SavingsGoal) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE savings_goals SET name = ?, target_amount_cents = ?, target_date = ?, target_wallet_id = ?, updated_at = ?
		WHERE id = ? AND user_email = ?
	`, goal.Name, goal.TargetAmountCents, nullableTime(goal.TargetDate), goal.TargetWalletID, goal.UpdatedAt.Format(time.RFC3339), goal.ID, goal.UserEmail)
	if err != nil {
		return fmt.Errorf("update goal: %w", err)
	}
	return nil
}

func (r *GoalRepoLibSQL) DeleteGoal(ctx context.Context, id, userEmail string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM savings_goals WHERE id = ? AND user_email = ?", id, userEmail)
	if err != nil {
		return fmt.Errorf("delete goal: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("goal not found")
	}
	return nil
}

func (r *GoalRepoLibSQL) FindGoalByID(ctx context.Context, id string) (*domain.SavingsGoal, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_email, name, target_amount_cents, target_date, target_wallet_id, created_at, updated_at
		FROM savings_goals WHERE id = ?
	`, id)
	return scanGoal(row)
}

func (r *GoalRepoLibSQL) ListGoals(ctx context.Context, userEmail string) ([]*domain.SavingsGoal, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_email, name, target_amount_cents, target_date, target_wallet_id, created_at, updated_at
		FROM savings_goals WHERE user_email = ? ORDER BY created_at DESC
	`, userEmail)
	if err != nil {
		return nil, fmt.Errorf("list goals: %w", err)
	}
	defer rows.Close()

	var goals []*domain.SavingsGoal
	for rows.Next() {
		g, err := scanGoal(rows)
		if err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, rows.Err()
}

func (r *GoalRepoLibSQL) SaveContribution(ctx context.Context, c *domain.GoalContribution) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO goal_contributions (id, goal_id, amount_cents, contributed_at, note, source_wallet_id, transfer_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, c.ID, c.GoalID, c.AmountCents, c.ContributedAt.Format("2006-01-02"), c.Note, nullableString(c.SourceWalletID), nullableString(c.TransferID), c.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("save contribution: %w", err)
	}
	return nil
}

func (r *GoalRepoLibSQL) ListContributionsByGoal(ctx context.Context, goalID string) ([]*domain.GoalContribution, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, goal_id, amount_cents, contributed_at, note, source_wallet_id, transfer_id, created_at
		FROM goal_contributions WHERE goal_id = ? ORDER BY contributed_at DESC
	`, goalID)
	if err != nil {
		return nil, fmt.Errorf("list contributions: %w", err)
	}
	defer rows.Close()

	var contributions []*domain.GoalContribution
	for rows.Next() {
		c, err := scanContribution(rows)
		if err != nil {
			return nil, err
		}
		contributions = append(contributions, c)
	}
	return contributions, rows.Err()
}

func (r *GoalRepoLibSQL) GetCurrentAmountByGoal(ctx context.Context, goalID string) (int64, error) {
	var total sql.NullInt64
	err := r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount_cents), 0) FROM goal_contributions WHERE goal_id = ?`, goalID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("get current amount: %w", err)
	}
	return total.Int64, nil
}

func scanGoal(row scannable) (*domain.SavingsGoal, error) {
	var g domain.SavingsGoal
	var targetDate *string
	var createdAt, updatedAt string
	var targetWalletID *string

	err := row.Scan(&g.ID, &g.UserEmail, &g.Name, &g.TargetAmountCents, &targetDate, &targetWalletID, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("goal not found")
	}
	if err != nil {
		return nil, fmt.Errorf("scan goal: %w", err)
	}

	if targetWalletID != nil {
		g.TargetWalletID = *targetWalletID
	}

	if targetDate != nil && *targetDate != "" {
		parsed, _ := time.Parse("2006-01-02", *targetDate)
		g.TargetDate = &parsed
	}
	g.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	g.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &g, nil
}

func scanContribution(row scannable) (*domain.GoalContribution, error) {
	var c domain.GoalContribution
	var contributedAt, createdAt string
	var note *string

	err := row.Scan(&c.ID, &c.GoalID, &c.AmountCents, &contributedAt, &note, &c.SourceWalletID, &c.TransferID, &createdAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("contribution not found")
	}
	if err != nil {
		return nil, fmt.Errorf("scan contribution: %w", err)
	}

	c.ContributedAt, _ = time.Parse("2006-01-02", contributedAt)
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	c.Note = note

	return &c, nil
}
