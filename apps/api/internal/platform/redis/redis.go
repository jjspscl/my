package redis

import (
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

func NewClient(redisURL string) (*redis.Client, error) {
	// Parse redis://user:pass@host:port/db
	if !strings.HasPrefix(redisURL, "redis://") {
		return nil, fmt.Errorf("invalid redis url: %s", redisURL)
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	client := redis.NewClient(opts)
	return client, nil
}