package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	// Embed the tzdata database so time.LoadLocation works on minimal runtime
	// images (scratch, distroless) that ship no /usr/share/zoneinfo. Every
	// binary that loads config — cmd/api, cmd/mcp, cmd/migrate — depends on
	// MY_TIMEZONE resolving or boot fails.
	_ "time/tzdata"
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
	Timezone        string
	Location        *time.Location
	MCPEnabled      bool
	MCPToken        string
	MCPBind         string
	MCPPort         string
	MCPReadOnly     bool
	MagicLinkRate   int
	LLMEnabled      bool
	LLMMasterKey    string
	LLMCodexPath    string
}

// MCPAddr is the listen address for the dedicated MCP listener. It is separate
// from the dashboard listener so MCPBind genuinely restricts which interfaces
// can reach the MCP surface.
func (c *Config) MCPAddr() string {
	return net.JoinHostPort(c.MCPBind, c.MCPPort)
}

func Load() (*Config, error) {
	sessionTTL := defaultEnv("MY_SESSION_TTL", "168h")
	dur, err := time.ParseDuration(sessionTTL)
	if err != nil {
		return nil, err
	}
	mcpEnabled, err := boolEnv("MY_MCP_ENABLED", false)
	if err != nil {
		return nil, err
	}
	mcpReadOnly, err := boolEnv("MY_MCP_READONLY", false)
	if err != nil {
		return nil, err
	}
	mcpToken := os.Getenv("MY_MCP_TOKEN")
	if mcpEnabled && len(mcpToken) < 32 {
		return nil, fmt.Errorf("MY_MCP_TOKEN must be at least 32 characters when MY_MCP_ENABLED=true")
	}
	magicLinkRate, err := intEnv("MY_MAGIC_LINK_RATE", 6)
	if err != nil {
		return nil, err
	}
	llmEnabled, err := boolEnv("MY_LLM_ENABLED", false)
	if err != nil {
		return nil, err
	}
	// Required with no default: every request path uses this as the data
	// ownership key. An empty value would silently create a phantom tenant.
	userEmail := os.Getenv("MY_USER_EMAIL")
	if userEmail == "" {
		return nil, fmt.Errorf("MY_USER_EMAIL is required")
	}
	webURL := defaultEnv("MY_WEB_URL", "http://localhost:5173")
	secureCookies, err := secureCookies(webURL)
	if err != nil {
		return nil, err
	}

	// The user's financial calendar. All "today", month boundary, and date-range
	// aggregation must use this location so dates do not shift against the
	// server's UTC clock. Defaults to the user's home timezone.
	timezone := defaultEnv("MY_TIMEZONE", "Asia/Manila")
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("MY_TIMEZONE: %w", err)
	}

	return &Config{
		APIPort:         defaultEnv("MY_API_PORT", "8080"),
		DatabaseURL:     defaultEnv("MY_DATABASE_URL", "file:my_dev.db"),
		RedisURL:        defaultEnv("MY_REDIS_URL", "redis://localhost:6379"),
		UserEmail:       userEmail,
		SMTPHost:        defaultEnv("MY_SMTP_HOST", "localhost"),
		SMTPPort:        defaultEnv("MY_SMTP_PORT", "1025"),
		SMTPFrom:        defaultEnv("MY_SMTP_FROM", "my@localhost"),
		SMTPUser:        os.Getenv("MY_SMTP_USER"),
		SMTPPass:        os.Getenv("MY_SMTP_PASS"),
		SessionTTL:      dur,
		WebURL:          webURL,
		SecureCookies:   secureCookies,
		DefaultCurrency: defaultEnv("MY_DEFAULT_CURRENCY", "PHP"),
		Timezone:        timezone,
		Location:        location,
		MCPEnabled:      mcpEnabled,
		MCPToken:        mcpToken,
		MCPBind:         defaultEnv("MY_MCP_BIND", "127.0.0.1"),
		MCPPort:         defaultEnv("MY_MCP_PORT", "8081"),
		MCPReadOnly:     mcpReadOnly,
		MagicLinkRate:   magicLinkRate,
		LLMEnabled:      llmEnabled,
		LLMMasterKey:    os.Getenv("MY_LLM_MASTER_KEY"),
		LLMCodexPath:    defaultEnv("MY_LLM_CODEX_PATH", ""),
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

func boolEnv(key string, fallback bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s: %w", key, err)
	}
	return parsed, nil
}

func intEnv(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return parsed, nil
}
