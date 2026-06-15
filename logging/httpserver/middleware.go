package httpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/vlaner/go-backend-examples/logging/canonicallog"
	"github.com/vlaner/go-backend-examples/logging/xctx"
)

const RequestIDHeader = "X-Request-Id"

type statusResponseWriter struct {
	http.ResponseWriter

	statusCode  int
	bytes       int
	wroteHeader bool
}

func LoggerMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	logger = logger.With(slog.String("component", "http.middleware"))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, requestID := contextWithRequestID(r)
		w.Header().Set(RequestIDHeader, requestID)

		responseWriter := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		startedAt := time.Now()

		logger.InfoContext(ctx, "request started", slog.String("method", r.Method), slog.String("path", r.URL.Path))
		next.ServeHTTP(responseWriter, r.WithContext(ctx))
		logger.InfoContext(
			ctx,
			"request finished",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", responseWriter.statusCode),
			slog.Int("response_bytes", responseWriter.bytes),
			slog.Duration("duration", time.Since(startedAt)),
		)
	})
}

func CanonicalLoggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	logger = logger.With(slog.String("component", "http.canonical"))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, requestID := contextWithRequestID(r)
		ctx = canonicallog.WithBag(ctx, canonicallog.NewBag())
		w.Header().Set(RequestIDHeader, requestID)

		responseWriter := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		startedAt := time.Now()

		next.ServeHTTP(responseWriter, r.WithContext(ctx))

		canonicallog.Set(ctx, "http.method", r.Method)
		canonicallog.Set(ctx, "http.path", r.URL.Path)
		canonicallog.Set(ctx, "http.status", responseWriter.statusCode)
		canonicallog.Set(ctx, "http.response_bytes", responseWriter.bytes)
		canonicallog.Set(ctx, "http.duration", time.Since(startedAt))

		logger.InfoContext(ctx, "http/request", canonicallog.Attrs(ctx)...)
	})
}

func (w *statusResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}

	w.statusCode = statusCode
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *statusResponseWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	n, err := w.ResponseWriter.Write(data)
	w.bytes += n
	if err != nil {
		return n, fmt.Errorf("write response: %w", err)
	}

	return n, nil
}

func (w *statusResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func contextWithRequestID(r *http.Request) (context.Context, string) {
	requestID := r.Header.Get(RequestIDHeader)
	if requestID == "" {
		requestID = uuid.NewString()
	}

	return xctx.WithRequestID(r.Context(), requestID), requestID
}
