package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type backoffFunc func(attempt int) time.Duration

func noBackoff(int) time.Duration { return 0 }

func withRetry(ctx context.Context, maxAttempts int, backoff backoffFunc, fn func() error) error {
	if maxAttempts <= 0 {
		maxAttempts = defaultTxMaxAttempts
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}
		if !isRetryableTxError(err) || attempt == maxAttempts {
			return err
		}

		err = retrySleep(ctx, backoff, attempt)
		if err != nil {
			return fmt.Errorf("retry backoff: %w", err)
		}
	}

	return nil
}

func retrySleep(ctx context.Context, backoff backoffFunc, attempt int) error {
	d := backoff(attempt)
	if d <= 0 {
		return nil
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("retry backoff context: %w", ctx.Err())
	case <-time.After(d):
		return nil
	}
}

func isRetryableTxError(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == "40001" || pgErr.Code == "40P01"
}
