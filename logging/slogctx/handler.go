package slogctx

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vlaner/go-backend-examples/logging/xctx"
)

type Handler struct {
	next slog.Handler
}

func NewHandler(next slog.Handler) *Handler {
	return &Handler{next: next}
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	requestID, ok := xctx.RequestID(ctx)
	if ok {
		record.AddAttrs(slog.String("request_id", requestID))
	}

	err := h.next.Handle(ctx, record)
	if err != nil {
		return fmt.Errorf("handle log record: %w", err)
	}

	return nil
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{next: h.next.WithAttrs(attrs)}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{next: h.next.WithGroup(name)}
}
