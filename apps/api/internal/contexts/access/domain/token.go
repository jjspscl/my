package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

type MagicToken struct {
	Token     string
	Email     string
	ExpiresAt time.Time
	UsedAt    *time.Time
}

func NewMagicToken(email string, ttl time.Duration) (*MagicToken, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &MagicToken{
		Token:     hex.EncodeToString(b),
		Email:     email,
		ExpiresAt: time.Now().UTC().Add(ttl),
	}, nil
}

func (t *MagicToken) IsExpired() bool {
	return time.Now().UTC().After(t.ExpiresAt)
}

func (t *MagicToken) IsUsed() bool {
	return t.UsedAt != nil
}

func (t *MagicToken) MarkUsed() {
	now := time.Now().UTC()
	t.UsedAt = &now
}