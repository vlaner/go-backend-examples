package postgres

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
)

type LoggingQueryTracer struct {
	logger *slog.Logger
}

func NewLoggingQueryTracer(logger *slog.Logger) *LoggingQueryTracer {
	return &LoggingQueryTracer{logger: logger}
}

func (t *LoggingQueryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, queryTraceKey{}, queryTrace{sql: data.SQL, startedAt: time.Now()})
}

func (t *LoggingQueryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	trace, _ := ctx.Value(queryTraceKey{}).(queryTrace)
	attrs := []any{"sql", trace.sql, "command_tag", data.CommandTag.String(), "err", data.Err}
	if !trace.startedAt.IsZero() {
		attrs = append(attrs, "duration", time.Since(trace.startedAt))
	}

	t.logger.InfoContext(ctx, "postgres query", attrs...)
}

type queryTraceKey struct{}

type queryTrace struct {
	sql       string
	startedAt time.Time
}
