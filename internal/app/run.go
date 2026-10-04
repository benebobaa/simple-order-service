package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/benebobaa/simple-order-service/db/migrations"
	"github.com/benebobaa/simple-order-service/internal/config"
)

const (
	startupTimeout  = 10 * time.Second
	shutdownTimeout = 10 * time.Second
)

// Run boots the service and blocks until the process receives SIGINT/SIGTERM
// or the HTTP server fails. cmd/api deliberately delegates the whole lifecycle
// here so main stays a tiny entrypoint.
func Run() error {
	cfg, logger, err := configure()
	if err != nil {
		return err
	}

	if cfg.AutoMigrate {
		logger.Info("applying database migrations")
		if err := migrations.Up(cfg.DatabaseURL); err != nil {
			return err
		}
	}

	pool, err := openDatabasePool(cfg)
	if err != nil {
		return err
	}
	defer pool.Close()

	return serve(cfg.Port, New(pool, cfg, logger), logger)
}

// configure loads configuration and installs the application logger. A
// bootstrap logger is created first so configuration failures are visible
// before the configured log level is known.
func configure() (*config.Config, *slog.Logger, error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		return nil, nil, err
	}

	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)
	return cfg, logger, nil
}

// openDatabasePool connects to PostgreSQL, applies optional pool tuning from
// the environment and verifies the connection with a bounded startup ping, so
// a dead database fails startup quickly.
func openDatabasePool(cfg *config.Config) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), startupTimeout)
	defer cancel()

	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	if cfg.DBMaxConns > 0 {
		poolCfg.MaxConns = cfg.DBMaxConns
	}
	if cfg.DBMinConns > 0 {
		poolCfg.MinConns = cfg.DBMinConns
	}
	if cfg.DBMaxConnLifetime > 0 {
		poolCfg.MaxConnLifetime = cfg.DBMaxConnLifetime
	}
	if cfg.DBMaxConnIdleTime > 0 {
		poolCfg.MaxConnIdleTime = cfg.DBMaxConnIdleTime
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

// serve runs the HTTP server and shuts it down gracefully on SIGINT/SIGTERM.
func serve(port string, handler http.Handler, logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server started", "addr", srv.Addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
