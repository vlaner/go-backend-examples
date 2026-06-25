package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
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
	return TxManager{pool: pool, config: DefaultTx()}
}

func (m TxManager) WithConfig(config TxConfig) TxManager {
	return TxManager{pool: m.pool, config: config}
}

func (m TxManager) Do(ctx context.Context, fn func(context.Context) error) error {
	err := withRetry(ctx, m.config.MaxAttempts, noBackoff, func() error {
		return m.runAttempt(ctx, fn)
	})
	if err != nil {
		return fmt.Errorf("postgres tx attempt: %w", err)
	}

	return nil
}

func (m TxManager) runAttempt(ctx context.Context, fn func(context.Context) error) error {
	err := pgx.BeginTxFunc(ctx, m.pool, m.config.Options, func(tx pgx.Tx) error {
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
