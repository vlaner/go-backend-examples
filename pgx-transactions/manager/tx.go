package manager

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vlaner/go-backend-examples/pgx-transactions/postgres"
)

type TxManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

func Executor(ctx context.Context, pool *pgxpool.Pool) postgres.DBTX {
	return postgres.ExecutorFromContext(ctx, pool)
}

type pgxTxManager struct {
	pool *pgxpool.Pool
}

func NewPGXManager(pool *pgxpool.Pool) TxManager {
	return &pgxTxManager{pool: pool}
}

func (m *pgxTxManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	err := pgx.BeginFunc(ctx, m.pool, func(tx pgx.Tx) error {
		txCtx := postgres.ContextWithTx(ctx, tx)
		return fn(txCtx)
	})
	if err != nil {
		return fmt.Errorf("pgx tx: %w", err)
	}

	return nil
}
