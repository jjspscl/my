package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jjspscl/my/internal/contexts/habits/domain"
)

type HabitRepoLibSQL struct {
	db *sql.DB
}

func NewHabitRepoLibSQL(db *sql.DB) *HabitRepoLibSQL {
	return &HabitRepoLibSQL{db: db}
}

func (r *HabitRepoLibSQL) SaveHabit(ctx context.Context, h *domain.Habit) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO habits (id, user_email, name, color, frequency, target_per_week, archived, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		h.ID, h.UserEmail, h.Name, h.Color, h.Frequency, h.TargetPerWeek, boolToInt(h.Archived), h.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("save habit: %w", err)
	}
	return nil
}

func (r *HabitRepoLibSQL) ListActive(ctx context.Context, userEmail string) ([]*domain.Habit, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_email, name, color, frequency, target_per_week, archived, created_at
		 FROM habits WHERE user_email = ? AND archived = 0
		 ORDER BY created_at DESC`, userEmail,
	)
	if err != nil {
		return nil, fmt.Errorf("list habits: %w", err)
	}
	defer rows.Close()

	var habits []*domain.Habit
	for rows.Next() {
		h, err := scanHabit(rows)
		if err != nil {
			return nil, err
		}
		habits = append(habits, h)
	}
	return habits, rows.Err()
}

func (r *HabitRepoLibSQL) FindByID(ctx context.Context, id, userEmail string) (*domain.Habit, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_email, name, color, frequency, target_per_week, archived, created_at
		 FROM habits WHERE id = ? AND user_email = ?`, id, userEmail,
	)
	return scanHabit(row)
}

func (r *HabitRepoLibSQL) Archive(ctx context.Context, id, userEmail string) error {
	result, err := r.db.ExecContext(ctx,
		"UPDATE habits SET archived = 1 WHERE id = ? AND user_email = ?", id, userEmail,
	)
	if err != nil {
		return fmt.Errorf("archive habit: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("habit not found")
	}
	return nil
}

func (r *HabitRepoLibSQL) SaveCompletion(ctx context.Context, c *domain.HabitCompletion) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO habit_completions (id, habit_id, completed_date, created_at)
		 VALUES (?, ?, ?, ?)`,
		c.ID, c.HabitID, c.CompletedDate.Format("2006-01-02"), c.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("save completion: %w", err)
	}
	return nil
}

func (r *HabitRepoLibSQL) DeleteCompletion(ctx context.Context, habitID string, date time.Time) error {
	result, err := r.db.ExecContext(ctx,
		"DELETE FROM habit_completions WHERE habit_id = ? AND completed_date = ?",
		habitID, date.Format("2006-01-02"),
	)
	if err != nil {
		return fmt.Errorf("delete completion: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("completion not found")
	}
	return nil
}

func (r *HabitRepoLibSQL) GetCompletion(ctx context.Context, habitID string, date time.Time) (*domain.HabitCompletion, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, habit_id, completed_date, created_at
		 FROM habit_completions WHERE habit_id = ? AND completed_date = ?`,
		habitID, date.Format("2006-01-02"),
	)
	return scanCompletion(row)
}

func (r *HabitRepoLibSQL) GetCompletionsInRange(ctx context.Context, habitID string, from, to time.Time) ([]*domain.HabitCompletion, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, habit_id, completed_date, created_at
		 FROM habit_completions
		 WHERE habit_id = ? AND completed_date >= ? AND completed_date <= ?
		 ORDER BY completed_date DESC`, habitID, from.Format("2006-01-02"), to.Format("2006-01-02"),
	)
	if err != nil {
		return nil, fmt.Errorf("get completions: %w", err)
	}
	defer rows.Close()

	var comps []*domain.HabitCompletion
	for rows.Next() {
		c, err := scanCompletion(rows)
		if err != nil {
			return nil, err
		}
		comps = append(comps, c)
	}
	return comps, rows.Err()
}

func (r *HabitRepoLibSQL) GetAllCompletionsInRange(ctx context.Context, userEmail string, from, to time.Time) (map[string][]*domain.HabitCompletion, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT hc.id, hc.habit_id, hc.completed_date, hc.created_at
		 FROM habit_completions hc
		 JOIN habits h ON h.id = hc.habit_id
		 WHERE h.user_email = ? AND hc.completed_date >= ? AND hc.completed_date <= ?
		 ORDER BY hc.completed_date DESC`, userEmail, from.Format("2006-01-02"), to.Format("2006-01-02"),
	)
	if err != nil {
		return nil, fmt.Errorf("get all completions: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]*domain.HabitCompletion)
	for rows.Next() {
		c, err := scanCompletion(rows)
		if err != nil {
			return nil, err
		}
		result[c.HabitID] = append(result[c.HabitID], c)
	}
	return result, rows.Err()
}

// ----- scan helpers -----

type scannable interface {
	Scan(dest ...any) error
}

func scanHabit(row scannable) (*domain.Habit, error) {
	var h domain.Habit
	var archivedInt int
	var createdAtStr string

	if err := row.Scan(&h.ID, &h.UserEmail, &h.Name, &h.Color, &h.Frequency,
		&h.TargetPerWeek, &archivedInt, &createdAtStr); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("habit not found")
		}
		return nil, fmt.Errorf("scan habit: %w", err)
	}

	h.Archived = archivedInt != 0
	createdAt, err := parseDatetimeHabit(createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	h.CreatedAt = createdAt

	return &h, nil
}

func scanCompletion(row scannable) (*domain.HabitCompletion, error) {
	var c domain.HabitCompletion
	var dateStr string
	var createdAtStr string

	if err := row.Scan(&c.ID, &c.HabitID, &dateStr, &createdAtStr); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("completion not found")
		}
		return nil, fmt.Errorf("scan completion: %w", err)
	}

	date, err := parseDatetimeHabit(dateStr)
	if err != nil {
		return nil, fmt.Errorf("parse completed_date: %w", err)
	}
	c.CompletedDate = date

	createdAt, err := parseDatetimeHabit(createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	c.CreatedAt = createdAt

	return &c, nil
}

func parseDatetimeHabit(s string) (time.Time, error) {
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

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
