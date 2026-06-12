package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type ConfigFunc func(*pgx.ConnConfig)

func WithTracer(tracer pgx.QueryTracer) ConfigFunc {
	return func(c *pgx.ConnConfig) {
		c.Tracer = tracer
	}
}

func Connect(ctx context.Context, dsn string, cfgOpts ...ConfigFunc) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse pool cfg: %w", err)
	}

	for _, o := range cfgOpts {
		o(poolConfig.ConnConfig)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("connect pgx: %w", err)
	}

	return pool, nil
}
