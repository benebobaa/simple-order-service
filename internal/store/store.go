// Package store provides database access built on top of sqlc-generated
// queries and manages transaction boundaries for the service layer.
package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/benebobaa/simple-order-service/internal/store/sqlc"
)

// Store bundles the connection pool with the sqlc query set.
type Store struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

// New creates a Store backed by the given pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, q: sqlc.New(pool)}
}

// Pool exposes the underlying connection pool.
func (s *Store) Pool() *pgxpool.Pool { return s.pool }

// Queries exposes the non-transactional query set.
func (s *Store) Queries() *sqlc.Queries { return s.q }

// WithTx runs fn inside a database transaction that is committed when fn
// returns nil and rolled back otherwise (including on panic).
func (s *Store) WithTx(ctx context.Context, fn func(q *sqlc.Queries) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(s.q.WithTx(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// IsUniqueViolation reports whether err is a PostgreSQL unique-constraint
// violation (SQLSTATE 23505).
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
