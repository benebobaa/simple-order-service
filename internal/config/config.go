// Package config loads application configuration from the environment.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration for the API.
type Config struct {
	AppEnv      string
	Port        string
	DatabaseURL string
	JWTSecret   string
	JWTTTL      time.Duration
	LogLevel    slog.Level
	AutoMigrate bool

	// Connection pool tuning. Zero means "use the pgx default"
	DBMaxConns        int32
	DBMinConns        int32
	DBMaxConnLifetime time.Duration
	DBMaxConnIdleTime time.Duration
}

// Load reads configuration from the process environment, loading a local
// .env file first when present. It fails fast when required values are missing
// so misconfiguration surfaces at startup rather than at first request.
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("load .env file: %w", err)
	}

	cfg := &Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	ttl, err := time.ParseDuration(getEnv("JWT_TTL", "24h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_TTL: %w", err)
	}
	cfg.JWTTTL = ttl

	level, err := parseLogLevel(getEnv("LOG_LEVEL", "info"))
	if err != nil {
		return nil, err
	}
	cfg.LogLevel = level

	autoMigrate, err := strconv.ParseBool(getEnv("AUTO_MIGRATE", "true"))
	if err != nil {
		return nil, fmt.Errorf("invalid AUTO_MIGRATE: %w", err)
	}
	cfg.AutoMigrate = autoMigrate

	if cfg.DBMaxConns, err = parseOptionalInt32("DB_MAX_CONNS"); err != nil {
		return nil, err
	}
	if cfg.DBMinConns, err = parseOptionalInt32("DB_MIN_CONNS"); err != nil {
		return nil, err
	}
	if cfg.DBMaxConnLifetime, err = parseOptionalDuration("DB_MAX_CONN_LIFETIME"); err != nil {
		return nil, err
	}
	if cfg.DBMaxConnIdleTime, err = parseOptionalDuration("DB_MAX_CONN_IDLE_TIME"); err != nil {
		return nil, err
	}
	if cfg.DBMinConns > 0 && cfg.DBMaxConns > 0 && cfg.DBMinConns > cfg.DBMaxConns {
		return nil, fmt.Errorf("DB_MIN_CONNS must not exceed DB_MAX_CONNS")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseLogLevel(s string) (slog.Level, error) {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid LOG_LEVEL %q (want debug|info|warn|error)", s)
	}
}

// parseOptionalInt32 parses an optional non-negative integer setting. An empty
// value means "not configured" and yields zero.
func parseOptionalInt32(key string) (int32, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid %s %q (want a non-negative integer)", key, raw)
	}
	return int32(n), nil
}

// parseOptionalDuration parses an optional positive duration setting. An empty
// value means "not configured" and yields zero.
func parseOptionalDuration(key string) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("invalid %s %q (want a positive duration like 1h or 30m)", key, raw)
	}
	return d, nil
}
