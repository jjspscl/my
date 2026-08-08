package config

import (
	"os"
	"strconv"
	"strings"
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
	WebURL          string
	SecureCookies   bool
	DefaultCurrency string
}

func Load() (*Config, error) {
	sessionTTL := defaultEnv("MY_SESSION_TTL", "168h")
	dur, err := time.ParseDuration(sessionTTL)
	if err != nil {
		return nil, err
	}
	webURL := defaultEnv("MY_WEB_URL", "http://localhost:5173")
	secureCookies, err := secureCookies(webURL)
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
		WebURL:          webURL,
		SecureCookies:   secureCookies,
		DefaultCurrency: defaultEnv("MY_DEFAULT_CURRENCY", "PHP"),
	}, nil
}

func secureCookies(webURL string) (bool, error) {
	if value := os.Getenv("MY_SECURE_COOKIES"); value != "" {
		secure, err := strconv.ParseBool(value)
		if err != nil {
			return false, err
		}
		return secure, nil
	}
	return strings.HasPrefix(strings.ToLower(webURL), "https://"), nil
}

func defaultEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
