package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jjspscl/my/internal/contexts/access/domain"
)

type TokenRepoLibSQL struct {
	db *sql.DB
}

func NewTokenRepoLibSQL(db *sql.DB) *TokenRepoLibSQL {
	return &TokenRepoLibSQL{db: db}
}

func (r *TokenRepoLibSQL) Save(ctx context.Context, token *domain.MagicToken) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO magic_tokens (token, email, expires_at, used_at) VALUES (?, ?, ?, ?)",
		token.Token, token.Email, token.ExpiresAt, nil,
	)
	if err != nil {
		return fmt.Errorf("save token: %w", err)
	}
	return nil
}

func (r *TokenRepoLibSQL) FindByToken(ctx context.Context, token string) (*domain.MagicToken, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT token, email, expires_at, used_at FROM magic_tokens WHERE token = ?",
		token,
	)

	var t domain.MagicToken
	var usedAt sql.Null[time.Time]
	if err := row.Scan(&t.Token, &t.Email, &t.ExpiresAt, &usedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("token not found")
		}
		return nil, fmt.Errorf("find token: %w", err)
	}

	if usedAt.Valid {
		t.UsedAt = &usedAt.V
	}

	return &t, nil
}

func (r *TokenRepoLibSQL) MarkUsed(ctx context.Context, token string, usedAt time.Time) error {
	result, err := r.db.ExecContext(ctx,
		"UPDATE magic_tokens SET used_at = ? WHERE token = ?",
		usedAt, token,
	)
	if err != nil {
		return fmt.Errorf("mark token used: %w", err)
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("token not found")
	}

	return nil
}
