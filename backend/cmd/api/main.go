// Command api boots the Nexo CRM backend: config, Postgres pool,
// versioned migrations and the HTTP server.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nexocrm/crm-saas/backend/internal/auth"
	"github.com/nexocrm/crm-saas/backend/internal/config"
	"github.com/nexocrm/crm-saas/backend/internal/db"
	"github.com/nexocrm/crm-saas/backend/internal/httpapi"
)

func main() {
	cfg := config.Load()
	setupLogger(cfg.LogLevel)

	if cfg.JWTSecret == "dev-secret-change-me" {
		slog.Warn("JWT_SECRET is the dev default; set a strong secret in production")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database unavailable", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if dir := resolveMigrationsDir(cfg.MigrationsDir); dir != "" {
		mctx, mcancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer mcancel()
		if err := db.Migrate(mctx, pool, dir); err != nil {
			slog.Error("migration failed", "error", err)
			os.Exit(1)
		}
	} else {
		slog.Warn("migrations directory not found; skipping auto-migrate")
	}

	server := httpapi.NewServer(
		pool,
		cfg.JWTSecret,
		cfg.AccessTTL,
		auth.NewRefreshStore(cfg.RefreshTTL),
		auth.NewResetStore(),
		cfg.CookieSecure,
	)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           server.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("api listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
}

func setupLogger(level string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})))
}

// resolveMigrationsDir locates ../migrations whether the binary runs from
// backend/ (go run) or from the repo root, honoring MIGRATIONS_DIR first.
func resolveMigrationsDir(explicit string) string {
	if explicit != "" {
		if st, err := os.Stat(explicit); err == nil && st.IsDir() {
			return explicit
		}
	}
	candidates := []string{"../migrations", "./migrations", "migrations", "backend/../migrations"}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	if exe, err := os.Executable(); err == nil {
		for _, c := range []string{
			filepath.Join(filepath.Dir(exe), "..", "migrations"),
			filepath.Join(filepath.Dir(exe), "migrations"),
		} {
			if st, err := os.Stat(c); err == nil && st.IsDir() {
				return c
			}
		}
	}
	return ""
}
