package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txContextKey struct{}

type contextExecutor struct {
	pool *pgxpool.Pool
}

func NewContextExecutor(pool *pgxpool.Pool) DBTX {
	return contextExecutor{pool: pool}
}

func ContextWithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

func ExecutorFromContext(ctx context.Context, pool *pgxpool.Pool) DBTX {
	tx, ok := ctx.Value(txContextKey{}).(pgx.Tx)
	if !ok {
		return pool
	}

	return tx
}

func (e contextExecutor) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	commandTag, err := ExecutorFromContext(ctx, e.pool).Exec(ctx, sql, args...)
	if err != nil {
		return commandTag, fmt.Errorf("exec postgres context executor: %w", err)
	}

	return commandTag, nil
}

func (e contextExecutor) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	rows, err := ExecutorFromContext(ctx, e.pool).Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query postgres context executor: %w", err)
	}

	return rows, nil
}

func (e contextExecutor) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return ExecutorFromContext(ctx, e.pool).QueryRow(ctx, sql, args...)
}
