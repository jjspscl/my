package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Store interface {
	Create(ctx context.Context, email string) (string, error)
	Get(ctx context.Context, sessionID string) (string, error)
	Delete(ctx context.Context, sessionID string) error
}

type RedisStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisStore(client *redis.Client, ttl time.Duration) *RedisStore {
	return &RedisStore{client: client, ttl: ttl}
}

func (s *RedisStore) Create(ctx context.Context, email string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}

	sessionID := hex.EncodeToString(b)
	key := fmt.Sprintf("session:%s", sessionID)

	if err := s.client.Set(ctx, key, email, s.ttl).Err(); err != nil {
		return "", fmt.Errorf("save session: %w", err)
	}

	return sessionID, nil
}

func (s *RedisStore) Get(ctx context.Context, sessionID string) (string, error) {
	key := fmt.Sprintf("session:%s", sessionID)
	email, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("session not found")
	}
	if err != nil {
		return "", fmt.Errorf("get session: %w", err)
	}
	// Sliding expiry: every authenticated request resets the TTL window, so
	// an active user is never bounced to email mid-use. Best-effort — a
	// failed refresh only shortens the session, it never breaks one.
	_ = s.client.Expire(ctx, key, s.ttl).Err()
	return email, nil
}

func (s *RedisStore) Delete(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return s.client.Del(ctx, key).Err()
}
