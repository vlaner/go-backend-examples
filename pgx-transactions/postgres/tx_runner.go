package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultTxMaxAttempts      = 1
	serializableTxMaxAttempts = 3
)

type TxConfig struct {
	Options     pgx.TxOptions
	MaxAttempts int
}

type TxManager struct {
	pool *pgxpool.Pool
}

type TxRunner struct {
	pool   *pgxpool.Pool
	config TxConfig
}

func DefaultTx() TxConfig {
	return TxConfig{MaxAttempts: defaultTxMaxAttempts}
}

func SerializableTx() TxConfig {
	return TxConfig{
		Options:     pgx.TxOptions{IsoLevel: pgx.Serializable},
		MaxAttempts: serializableTxMaxAttempts,
	}
}

func NewTxManager(pool *pgxpool.Pool) TxManager {
	return TxManager{pool: pool}
}

func (m TxManager) WithConfig(config TxConfig) TxRunner {
	return TxRunner{pool: m.pool, config: config}
}

func (r TxRunner) Do(ctx context.Context, fn func(context.Context) error) error {
	maxAttempts := r.config.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = defaultTxMaxAttempts
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := r.runAttempt(ctx, fn)
		if err == nil {
			return nil
		}
		if isRetryableTxError(err) && attempt < maxAttempts {
			continue
		}

		return fmt.Errorf("postgres tx attempt: %w", err)
	}

	return nil
}

func (r TxRunner) runAttempt(ctx context.Context, fn func(context.Context) error) error {
	err := pgx.BeginTxFunc(ctx, r.pool, r.config.Options, func(tx pgx.Tx) error {
		txCtx := ContextWithTx(ctx, tx)
		err := fn(txCtx)
		if err != nil {
			return fmt.Errorf("run tx function: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("pgx tx: %w", err)
	}

	return nil
}

func isRetryableTxError(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == "40001" || pgErr.Code == "40P01"
}
