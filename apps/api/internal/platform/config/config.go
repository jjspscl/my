package config

import (
	"os"
	"time"
)

type Config struct {
	APIPort         string
	DatabaseURL     string
	RedisURL        string
	UserEmail       string
	SMTPHost        string
	SMTPPort        string
	SMTPFrom        string
	SMTPUser        string
	SMTPPass        string
	SessionTTL      time.Duration
	CookieSecret    string
	CSRFSecret      string
	WebURL          string
	DefaultCurrency string
}

func Load() (*Config, error) {
	sessionTTL := defaultEnv("MY_SESSION_TTL", "168h")
	dur, err := time.ParseDuration(sessionTTL)
	if err != nil {
		return nil, err
	}

	return &Config{
		APIPort:         defaultEnv("MY_API_PORT", "8080"),
		DatabaseURL:     defaultEnv("MY_DATABASE_URL", "file:my_dev.db"),
		RedisURL:        defaultEnv("MY_REDIS_URL", "redis://localhost:6379"),
		UserEmail:       os.Getenv("MY_USER_EMAIL"),
		SMTPHost:        defaultEnv("MY_SMTP_HOST", "localhost"),
		SMTPPort:        defaultEnv("MY_SMTP_PORT", "1025"),
		SMTPFrom:        defaultEnv("MY_SMTP_FROM", "my@localhost"),
		SMTPUser:        os.Getenv("MY_SMTP_USER"),
		SMTPPass:        os.Getenv("MY_SMTP_PASS"),
		SessionTTL:      dur,
		CookieSecret:    os.Getenv("MY_COOKIE_SECRET"),
		CSRFSecret:      os.Getenv("MY_CSRF_SECRET"),
		WebURL:          defaultEnv("MY_WEB_URL", "http://localhost:5173"),
		DefaultCurrency: defaultEnv("MY_DEFAULT_CURRENCY", "PHP"),
	}, nil
}

func defaultEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
