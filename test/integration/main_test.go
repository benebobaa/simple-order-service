// Package integration contains full-stack tests that exercise the real HTTP
// router against a real PostgreSQL instance running in a throwaway container.
package integration

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/benebobaa/simple-order-service/db/migrations"
	"github.com/benebobaa/simple-order-service/internal/app"
	"github.com/benebobaa/simple-order-service/internal/config"
)

var (
	testPool   *pgxpool.Pool
	testRouter http.Handler
)

func TestMain(m *testing.M) {
	code, err := run(m)
	if err != nil {
		fmt.Fprintln(os.Stderr, "integration test setup failed:", err)
		os.Exit(1)
	}
	os.Exit(code)
}

func run(m *testing.M) (int, error) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("orders_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(2*time.Minute),
		),
	)
	if err != nil {
		return 0, fmt.Errorf("start postgres container: %w", err)
	}
	defer func() { _ = pgContainer.Terminate(ctx) }()

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return 0, fmt.Errorf("get connection string: %w", err)
	}

	if err := migrations.Up(dsn); err != nil {
		return 0, fmt.Errorf("apply migrations: %w", err)
	}

	testPool, err = pgxpool.New(ctx, dsn)
	if err != nil {
		return 0, fmt.Errorf("create connection pool: %w", err)
	}
	defer testPool.Close()

	cfg := &config.Config{
		AppEnv:    "test",
		JWTSecret: "integration-test-secret",
		JWTTTL:    time.Hour,
	}
	testRouter = app.New(testPool, cfg, slog.New(slog.DiscardHandler))

	return m.Run(), nil
}
