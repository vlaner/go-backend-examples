package postgres

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/vlaner/go-backend-examples/logging/canonicallog"
)

const (
	defaultLogKey = "db"
	Message       = "postgres/query"
)

//nolint:gochecknoglobals // Shared strings.Replacer avoids rebuilding it for every query log.
var spaceReplacer = strings.NewReplacer("\n", " ", "\t", "")

type DBLog struct {
	Operation    string
	Query        string
	Args         []any
	Duration     time.Duration
	Error        error
	RowsAffected int64
	LogKey       string
}

type Opts struct {
	LogKey     string
	RedactKeys []string
}

type MultiQueryTracer struct {
	tracers []pgx.QueryTracer
}

type LoggingQueryTracer struct {
	logger *slog.Logger
	opts   Opts
}

type CanonicalQueryTracer struct {
	opts Opts
}

type pgErrorWrapper struct {
	err error
}

type loggingQueryTraceKey struct{}

type canonicalQueryTraceKey struct{}

type queryTrace struct {
	sql       string
	args      []any
	startedAt time.Time
}

func NewMultiQueryTracer(tracers ...pgx.QueryTracer) *MultiQueryTracer {
	return &MultiQueryTracer{tracers: tracers}
}

func NewLoggingQueryTracer(logger *slog.Logger, opts Opts) *LoggingQueryTracer {
	if opts.LogKey == "" {
		opts.LogKey = defaultLogKey
	}

	return &LoggingQueryTracer{
		logger: logger.With(slog.String("component", "postgres.query")),
		opts:   opts,
	}
}

func NewCanonicalQueryTracer(opts Opts) *CanonicalQueryTracer {
	return &CanonicalQueryTracer{opts: opts}
}

func (t *MultiQueryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	for _, tracer := range t.tracers {
		//nolint:fatcontext // Tracer chaining must pass each tracer's returned context to the next tracer.
		ctx = tracer.TraceQueryStart(ctx, conn, data)
	}

	return ctx
}

func (t *MultiQueryTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	for _, tracer := range t.tracers {
		tracer.TraceQueryEnd(ctx, conn, data)
	}
}

func (d DBLog) LogValue() slog.Value {
	return slog.GroupValue(d.Attrs()...)
}

func (d DBLog) Attrs() []slog.Attr {
	attrs := []slog.Attr{
		slog.String("op", d.Operation),
		slog.Duration("duration", d.Duration),
		slog.String("sql", cleanSQL(d.Query)),
	}

	argAttr := convertArgs(d.Args)
	if argAttr.Key != "" {
		attrs = append(attrs, argAttr)
	}

	if d.Error != nil {
		attrs = append(attrs, slog.Any("error", pgErrorWrapper{err: d.Error}))
	}

	if d.RowsAffected > 0 {
		attrs = append(attrs, slog.Int64("rows", d.RowsAffected))
	}

	return attrs
}

func (w pgErrorWrapper) LogValue() slog.Value {
	if w.err == nil {
		return slog.GroupValue()
	}

	attrs := []slog.Attr{slog.String("message", w.err.Error())}

	var pgErr *pgconn.PgError
	if errors.As(w.err, &pgErr) {
		attrs = append(attrs,
			slog.String("code", pgErr.Code),
			slog.String("severity", pgErr.Severity),
			slog.String("table", pgErr.TableName),
			slog.String("constraint", pgErr.ConstraintName),
		)
		if pgErr.Detail != "" {
			attrs = append(attrs, slog.String("detail", pgErr.Detail))
		}
		if pgErr.Hint != "" {
			attrs = append(attrs, slog.String("hint", pgErr.Hint))
		}

		return slog.GroupValue(attrs...)
	}

	if errors.Is(w.err, context.DeadlineExceeded) {
		attrs = append(attrs, slog.String("type", "timeout"))
	} else if errors.Is(w.err, context.Canceled) {
		attrs = append(attrs, slog.String("type", "canceled"))
	}

	return slog.GroupValue(attrs...)
}

func (t *LoggingQueryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, loggingQueryTraceKey{}, queryTrace{
		sql:       data.SQL,
		args:      data.Args,
		startedAt: time.Now(),
	})
}

func (t *LoggingQueryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	trace, _ := ctx.Value(loggingQueryTraceKey{}).(queryTrace)
	dbLog := DBLog{
		Operation:    "query",
		Query:        trace.sql,
		Args:         redactArgs(trace.args, t.opts.RedactKeys),
		Duration:     time.Since(trace.startedAt),
		Error:        data.Err,
		RowsAffected: data.CommandTag.RowsAffected(),
		LogKey:       t.opts.LogKey,
	}

	t.logger.InfoContext(ctx, Message, slog.Any(dbLog.LogKey, dbLog))
}

func (t *CanonicalQueryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, canonicalQueryTraceKey{}, queryTrace{
		sql:       data.SQL,
		args:      data.Args,
		startedAt: time.Now(),
	})
}

func (t *CanonicalQueryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	trace, _ := ctx.Value(canonicalQueryTraceKey{}).(queryTrace)
	duration := time.Since(trace.startedAt)

	rowsAffected := data.CommandTag.RowsAffected()
	dbLog := DBLog{
		Operation:    "query",
		Query:        trace.sql,
		Args:         redactArgs(trace.args, t.opts.RedactKeys),
		Duration:     duration,
		Error:        data.Err,
		RowsAffected: rowsAffected,
		LogKey:       defaultLogKey,
	}

	canonicallog.AppendGroup(ctx, "db.queries", dbLog.Attrs()...)
	canonicallog.Add(ctx, "db.query_count", 1)
	canonicallog.AddDuration(ctx, "db.duration", duration)

	if rowsAffected > 0 {
		canonicallog.Add(ctx, "db.rows_affected", rowsAffected)
	}

	if data.Err == nil {
		return
	}

	canonicallog.Add(ctx, "db.error_count", 1)
}

func cleanSQL(sql string) string {
	return strings.TrimSpace(spaceReplacer.Replace(sql))
}

func convertArgs(args []any) slog.Attr {
	if len(args) == 0 {
		return slog.Attr{}
	}
	if len(args) == 1 {
		if namedArgs, ok := args[0].(pgx.NamedArgs); ok {
			return slog.Any("args", map[string]any(namedArgs))
		}
	}

	return slog.Int("args_count", len(args))
}

func redactArgs(args []any, redactKeys []string) []any {
	if len(redactKeys) == 0 || len(args) != 1 {
		return args
	}

	namedArgs, ok := args[0].(pgx.NamedArgs)
	if !ok {
		return args
	}

	return []any{redactNamedArgs(namedArgs, redactKeys)}
}

func redactNamedArgs(args pgx.NamedArgs, redactKeys []string) pgx.NamedArgs {
	redacted := make(pgx.NamedArgs, len(args))
	for key, val := range args {
		if containsRedactKey(redactKeys, key) {
			redacted[key] = "[REDACTED]"
			continue
		}

		redacted[key] = val
	}

	return redacted
}

func containsRedactKey(redactKeys []string, key string) bool {
	for _, redactKey := range redactKeys {
		if redactKey == "" {
			continue
		}

		if redactKey == key {
			return true
		}
	}

	return false
}
