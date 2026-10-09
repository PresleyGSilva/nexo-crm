// Package config loads service configuration from the environment.
package config

import (
	"os"
	"time"
)

// Config holds all runtime settings for the API server.
type Config struct {
	Port          string
	DatabaseURL   string
	JWTSecret     string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
	CookieSecure  bool
	MigrationsDir string
	LogLevel      string
}

// Load reads configuration from environment variables with sane local defaults.
func Load() Config {
	return Config{
		Port:          getenv("PORT", "8080"),
		DatabaseURL:   getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/crm?sslmode=disable"),
		JWTSecret:     getenv("JWT_SECRET", "dev-secret-change-me"),
		AccessTTL:     getenvDuration("JWT_ACCESS_TTL", 15*time.Minute),
		RefreshTTL:    getenvDuration("JWT_REFRESH_TTL", 168*time.Hour),
		CookieSecure:  getenv("COOKIE_SECURE", "false") == "true",
		MigrationsDir: os.Getenv("MIGRATIONS_DIR"),
		LogLevel:      getenv("LOG_LEVEL", "info"),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
