package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
)

type CategoryRepoLibSQL struct {
	db *sql.DB
}

func NewCategoryRepoLibSQL(db *sql.DB) *CategoryRepoLibSQL {
	return &CategoryRepoLibSQL{db: db}
}

func (r *CategoryRepoLibSQL) List(ctx context.Context) ([]*domain.Category, error) {
	rows, err := executor(ctx, r.db).QueryContext(ctx, `
		SELECT name, classification, essential, active, created_at, updated_at
		FROM finance_categories
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var categories []*domain.Category
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

func (r *CategoryRepoLibSQL) FindByName(ctx context.Context, name string) (*domain.Category, error) {
	row := executor(ctx, r.db).QueryRowContext(ctx, `
		SELECT name, classification, essential, active, created_at, updated_at
		FROM finance_categories WHERE name = ?
	`, name)
	return scanCategory(row)
}

func (r *CategoryRepoLibSQL) Update(ctx context.Context, category *domain.Category) error {
	res, err := executor(ctx, r.db).ExecContext(ctx, `
		UPDATE finance_categories
		SET classification = ?, essential = ?, active = ?, updated_at = ?
		WHERE name = ?
	`, category.Classification, boolToInt(category.Essential), boolToInt(category.Active),
		category.UpdatedAt.Format(time.RFC3339), category.Name)
	if err != nil {
		return fmt.Errorf("update category: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update category rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("category not found")
	}
	return nil
}

func scanCategory(row scannable) (*domain.Category, error) {
	var c domain.Category
	var essential, active int
	var createdAt, updatedAt string

	err := row.Scan(&c.Name, &c.Classification, &essential, &active, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("category not found")
	}
	if err != nil {
		return nil, fmt.Errorf("scan category: %w", err)
	}

	c.Essential = essential == 1
	c.Active = active == 1
	c.CreatedAt, _ = parseDatetime(createdAt)
	c.UpdatedAt, _ = parseDatetime(updatedAt)

	return &c, nil
}
