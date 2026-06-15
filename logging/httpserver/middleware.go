package httpserver

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/vlaner/go-backend-examples/logging/xctx"
)

const RequestIDHeader = "X-Request-Id"

type statusResponseWriter struct {
	http.ResponseWriter

	statusCode int
}

func LoggerMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	logger = logger.With(slog.String("component", "http.middleware"))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.NewString()
		}

		ctx := xctx.WithRequestID(r.Context(), requestID)
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
			slog.Duration("duration", time.Since(startedAt)),
		)
	})
}

func (w *statusResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
