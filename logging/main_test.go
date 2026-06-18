package main

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/multitracer"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	pgcontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/vlaner/go-backend-examples/logging/httpserver"
	loggingpostgres "github.com/vlaner/go-backend-examples/logging/postgres"
	"github.com/vlaner/go-backend-examples/logging/slogctx"
	"github.com/vlaner/go-backend-examples/logging/userhandler"
	"github.com/vlaner/go-backend-examples/logging/userrepo"
	"github.com/vlaner/go-backend-examples/logging/userservice"
)

func TestCreateUserLogsRequestAndPostgresQuery(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	var logs bytes.Buffer
	logger := slog.New(slogctx.NewHandler(slogctx.NewMultilineTextHandler(&logs)))
	t.Cleanup(func() {
		t.Logf("logger output:\n%s", logs.String())
	})

	pool := newTestPool(ctx, t, logger)
	handler := newTestHandler(pool, logger)

	response := postUser(handler, "req-123")
	if response.Code != http.StatusCreated {
		t.Fatalf("response status = %d, want %d", response.Code, http.StatusCreated)
	}
	if response.Header().Get(httpserver.RequestIDHeader) != "req-123" {
		t.Fatalf("response request id = %q, want %q", response.Header().Get(httpserver.RequestIDHeader), "req-123")
	}
	if !strings.Contains(response.Body.String(), `"username":"alice"`) {
		t.Fatalf("response body = %q, want username alice", response.Body.String())
	}

	duplicateResponse := postUser(handler, "req-duplicate")
	if duplicateResponse.Code != http.StatusInternalServerError {
		t.Fatalf("duplicate response status = %d, want %d", duplicateResponse.Code, http.StatusInternalServerError)
	}

	assertLogOutput(t, logs.String())
}

func newTestPool(ctx context.Context, t *testing.T, logger *slog.Logger) *pgxpool.Pool {
	t.Helper()

	container, err := pgcontainer.Run(ctx,
		"postgres:18-alpine3.23",
		pgcontainer.WithDatabase("app"),
		pgcontainer.WithUsername("app"),
		pgcontainer.WithPassword("app"),
		pgcontainer.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("run postgres container: %v", err)
	}
	t.Cleanup(func() {
		terminateErr := testcontainers.TerminateContainer(container)
		if terminateErr != nil {
			t.Fatalf("terminate postgres container: %v", terminateErr)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("postgres connection string: %v", err)
	}

	redactKeys := loggingpostgres.RedactKeys("password")
	tracer := multitracer.New(
		loggingpostgres.NewLoggingQueryTracer(logger, loggingpostgres.Opts{RedactKeys: redactKeys}),
		loggingpostgres.NewCanonicalQueryTracer(loggingpostgres.Opts{RedactKeys: redactKeys}),
	)
	pool, err := loggingpostgres.Connect(ctx, dsn, loggingpostgres.WithTracer(tracer))
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(pool.Close)

	err = loggingpostgres.Migrate(ctx, pool)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return pool
}

func newTestHandler(pool *pgxpool.Pool, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	repository := userrepo.New(pool)
	service := userservice.New(repository, logger)
	canonicalService := userservice.NewCanonicalCreateUser(service)
	handler := userhandler.New(canonicalService, logger)
	handler.RegisterRoutes(mux)

	return httpserver.CanonicalLoggingMiddleware(logger, mux)
}

func postUser(handler http.Handler, requestID string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"username":"alice","password":"secret"}`))
	request.Header.Set(httpserver.RequestIDHeader, requestID)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	return response
}

func assertLogOutput(t *testing.T, output string) {
	t.Helper()

	if !strings.Contains(output, "msg=postgres/query") {
		t.Fatalf("logs do not contain postgres query: %s", output)
	}
	if !strings.Contains(output, "request_id=req-123") {
		t.Fatalf("logs do not contain request id: %s", output)
	}
	if !strings.Contains(output, "request_id=req-duplicate") {
		t.Fatalf("logs do not contain duplicate request id: %s", output)
	}
	if !strings.Contains(output, "db.error.code=23505") {
		t.Fatalf("logs do not contain postgres query unique violation code: %s", output)
	}
	if !strings.Contains(output, "msg=http/request") {
		t.Fatalf("logs do not contain canonical request log: %s", output)
	}
	if !strings.Contains(output, "db.queries.0.error.code=23505") {
		t.Fatalf("canonical log does not contain unique violation code on query entry: %s", output)
	}
	if !strings.Contains(output, "db.query_count=2") {
		t.Fatalf("canonical log does not contain successful request db query count: %s", output)
	}
	if !strings.Contains(output, "db.query_count=1") {
		t.Fatalf("canonical log does not contain failed request db query count: %s", output)
	}
	if !strings.Contains(output, "db.queries.0.sql=") {
		t.Fatalf("canonical log does not contain first query entry sql: %s", output)
	}
	if !strings.Contains(output, "db.queries.1.sql=") {
		t.Fatalf("canonical log does not contain second query entry sql: %s", output)
	}
	if !strings.Contains(output, "service.operation=create_user") {
		t.Fatalf("canonical log does not contain service operation: %s", output)
	}
	if !strings.Contains(output, "service.outcome=created") {
		t.Fatalf("canonical log does not contain created service outcome: %s", output)
	}
	if !strings.Contains(output, "service.outcome=failed") {
		t.Fatalf("canonical log does not contain failed service outcome: %s", output)
	}
	if !strings.Contains(output, "service.user_id=") {
		t.Fatalf("canonical log does not contain created user id: %s", output)
	}
	if !strings.Contains(output, "[REDACTED]") {
		t.Fatalf("logs do not contain redacted fields: %s", output)
	}
	if strings.Contains(output, "secret") {
		t.Fatalf("logs contain raw password: %s", output)
	}
}
