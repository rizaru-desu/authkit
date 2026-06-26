// Package config loads application configuration from environment variables.
package config

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration.
type Config struct {
	App       AppConfig
	Server    ServerConfig
	Database  DatabaseConfig
	JWT       JWTConfig
	Session   SessionConfig
	RateLimit RateLimitConfig
	Email     EmailConfig
	// TrustedOrigins are allowed CORS origins / CSRF origin allowlist
	// (Better Auth style). Supports wildcards: *, **, ?.
	TrustedOrigins []string
	// RequireEmailVerification blocks sign-in until the email is verified.
	RequireEmailVerification bool
}

type EmailConfig struct {
	SMTPHost    string
	SMTPPort    int
	SMTPUser    string
	SMTPPass    string
	FromAddress string
	FromName    string
	AppBaseURL  string // frontend base URL used to build verification/reset links
}

type SessionConfig struct {
	Expiry time.Duration
	// Cookie transport (for web clients). Bearer header is always supported.
	CookieName     string
	CookieDomain   string
	CookieSecure   bool
	CookieSameSite http.SameSite
}

type RateLimitConfig struct {
	Window time.Duration
	Max    int
}

type AppConfig struct {
	Name string
	Env  string
}

type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DatabaseConfig struct {
	URL          string
	MaxOpenConns int
	MaxIdleConns int
	ConnLifetime time.Duration
}

type JWTConfig struct {
	Secret        string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	dbMaxOpen, err := strconv.Atoi(getEnv("DB_MAX_OPEN_CONNS", "25"))
	if err != nil {
		return nil, fmt.Errorf("DB_MAX_OPEN_CONNS: %w", err)
	}
	dbMaxIdle, err := strconv.Atoi(getEnv("DB_MAX_IDLE_CONNS", "5"))
	if err != nil {
		return nil, fmt.Errorf("DB_MAX_IDLE_CONNS: %w", err)
	}
	serverPort, err := strconv.Atoi(getEnv("SERVER_PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("SERVER_PORT: %w", err)
	}
	readTimeout, err := time.ParseDuration(getEnv("SERVER_READ_TIMEOUT", "10s"))
	if err != nil {
		return nil, fmt.Errorf("SERVER_READ_TIMEOUT: %w", err)
	}
	writeTimeout, err := time.ParseDuration(getEnv("SERVER_WRITE_TIMEOUT", "10s"))
	if err != nil {
		return nil, fmt.Errorf("SERVER_WRITE_TIMEOUT: %w", err)
	}
	connLifetime, err := time.ParseDuration(getEnv("DB_CONN_LIFETIME", "5m"))
	if err != nil {
		return nil, fmt.Errorf("DB_CONN_LIFETIME: %w", err)
	}
	accessExpiry, err := time.ParseDuration(getEnv("JWT_ACCESS_EXPIRY", "15m"))
	if err != nil {
		return nil, fmt.Errorf("JWT_ACCESS_EXPIRY: %w", err)
	}
	refreshExpiry, err := time.ParseDuration(getEnv("JWT_REFRESH_EXPIRY", "168h"))
	if err != nil {
		return nil, fmt.Errorf("JWT_REFRESH_EXPIRY: %w", err)
	}
	sessionExpiry, err := time.ParseDuration(getEnv("SESSION_EXPIRY", "168h"))
	if err != nil {
		return nil, fmt.Errorf("SESSION_EXPIRY: %w", err)
	}
	cookieSecure, err := strconv.ParseBool(getEnv("SESSION_COOKIE_SECURE", "false"))
	if err != nil {
		return nil, fmt.Errorf("SESSION_COOKIE_SECURE: %w", err)
	}
	sameSite, err := parseSameSite(getEnv("SESSION_COOKIE_SAMESITE", "lax"))
	if err != nil {
		return nil, err
	}
	smtpPort, err := strconv.Atoi(getEnv("EMAIL_SMTP_PORT", "587"))
	if err != nil {
		return nil, fmt.Errorf("EMAIL_SMTP_PORT: %w", err)
	}
	rateWindow, err := time.ParseDuration(getEnv("RATE_LIMIT_WINDOW", "60s"))
	if err != nil {
		return nil, fmt.Errorf("RATE_LIMIT_WINDOW: %w", err)
	}
	rateMax, err := strconv.Atoi(getEnv("RATE_LIMIT_MAX", "100"))
	if err != nil {
		return nil, fmt.Errorf("RATE_LIMIT_MAX: %w", err)
	}
	requireEmailVerification, err := strconv.ParseBool(getEnv("REQUIRE_EMAIL_VERIFICATION", "false"))
	if err != nil {
		return nil, fmt.Errorf("REQUIRE_EMAIL_VERIFICATION: %w", err)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return &Config{
		App: AppConfig{
			Name: getEnv("APP_NAME", "mns-backend"),
			Env:  getEnv("APP_ENV", "development"),
		},
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			Port:         serverPort,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
		},
		Database: DatabaseConfig{
			URL:          dbURL,
			MaxOpenConns: dbMaxOpen,
			MaxIdleConns: dbMaxIdle,
			ConnLifetime: connLifetime,
		},
		JWT: JWTConfig{
			Secret:        jwtSecret,
			AccessExpiry:  accessExpiry,
			RefreshExpiry: refreshExpiry,
		},
		Session: SessionConfig{
			Expiry:         sessionExpiry,
			CookieName:     getEnv("SESSION_COOKIE_NAME", "mns_session"),
			CookieDomain:   getEnv("SESSION_COOKIE_DOMAIN", ""),
			CookieSecure:   cookieSecure,
			CookieSameSite: sameSite,
		},
		RateLimit: RateLimitConfig{
			Window: rateWindow,
			Max:    rateMax,
		},
		Email: EmailConfig{
			SMTPHost:    getEnv("EMAIL_SMTP_HOST", ""),
			SMTPPort:    smtpPort,
			SMTPUser:    getEnv("EMAIL_SMTP_USER", ""),
			SMTPPass:    getEnv("EMAIL_SMTP_PASS", ""),
			FromAddress: getEnv("EMAIL_FROM_ADDRESS", "no-reply@example.com"),
			FromName:    getEnv("EMAIL_FROM_NAME", getEnv("APP_NAME", "mns-backend")),
			AppBaseURL:  getEnv("APP_BASE_URL", "http://localhost:3000"),
		},
		TrustedOrigins:           splitAndTrim(getEnv("TRUSTED_ORIGINS", "")),
		RequireEmailVerification: requireEmailVerification,
	}, nil
}

// parseSameSite maps a config string to http.SameSite. "none" requires Secure.
func parseSameSite(s string) (http.SameSite, error) {
	switch strings.ToLower(s) {
	case "lax":
		return http.SameSiteLaxMode, nil
	case "strict":
		return http.SameSiteStrictMode, nil
	case "none":
		return http.SameSiteNoneMode, nil
	default:
		return 0, fmt.Errorf("SESSION_COOKIE_SAMESITE: invalid value %q (use lax|strict|none)", s)
	}
}

// splitAndTrim splits a comma-separated list, trimming spaces and dropping empties.
func splitAndTrim(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
