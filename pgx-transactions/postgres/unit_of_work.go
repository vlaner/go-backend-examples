package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vlaner/go-backend-examples/pgx-transactions/unitofwork"
)

type pgxUnitOfWork[T any] struct {
	pool     *pgxpool.Pool
	newValue func(db DBTX) T
}

func NewPGXUnitOfWork[T any](pool *pgxpool.Pool, newValue func(db DBTX) T) unitofwork.UnitOfWork[T] {
	return &pgxUnitOfWork[T]{
		pool:     pool,
		newValue: newValue,
	}
}

func (u *pgxUnitOfWork[T]) Do(ctx context.Context, fn func(ctx context.Context, value T) error) error {
	err := pgx.BeginFunc(ctx, u.pool, func(tx pgx.Tx) error {
		value := u.newValue(tx)
		return fn(ctx, value)
	})
	if err != nil {
		return fmt.Errorf("pgx unit of work: %w", err)
	}

	return nil
}
