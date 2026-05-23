package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New creates a structured JSON logger. Level from MY_LOG_LEVEL env (default: info).
func New() *slog.Logger {
	level := parseLevel(os.Getenv("MY_LOG_LEVEL"))
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	return slog.New(handler)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
