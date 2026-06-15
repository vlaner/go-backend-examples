package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context, dsn string, logger *slog.Logger) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse pool config: %w", err)
	}

	config.ConnConfig.Tracer = NewLoggingQueryTracer(logger, Opts{RedactKeys: []string{"password"}})

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("connect pgx pool: %w", err)
	}

	return pool, nil
}

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("migrate logging example: %w", err)
	}

	return nil
}
