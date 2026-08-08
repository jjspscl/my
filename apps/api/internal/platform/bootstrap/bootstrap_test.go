package bootstrap

import (
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/jjspscl/my/internal/platform/config"
)

func TestNewWiresApplicationServices(t *testing.T) {
	if !redisAvailable(t) {
		t.Skip("Redis unavailable; bootstrap integration test requires Redis")
	}

	app, err := New(&config.Config{
		DatabaseURL:     "file:" + t.TempDir() + "/bootstrap.db",
		RedisURL:        "redis://localhost:6379",
		SessionTTL:      time.Hour,
		DefaultCurrency: "PHP",
		UserEmail:       "user@example.com",
	}, slog.Default())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	for name, value := range map[string]any{
		"auth": app.Auth, "transactions": app.Tx, "budget": app.Budget,
		"bills": app.Bill, "goals": app.Goal, "wallets": app.Wallet,
		"transfers": app.Transfer, "habits": app.Habit,
	} {
		if value == nil {
			t.Errorf("%s service is nil", name)
		}
	}

	if err := app.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := app.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
}

func redisAvailable(t *testing.T) bool {
	t.Helper()
	conn, err := net.DialTimeout("tcp", "127.0.0.1:6379", time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
