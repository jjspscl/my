package domain

import (
	"context"
	"time"
)

type TokenRepository interface {
	Save(ctx context.Context, token *MagicToken) error
	FindByToken(ctx context.Context, token string) (*MagicToken, error)
	MarkUsed(ctx context.Context, token string, usedAt time.Time) error
}