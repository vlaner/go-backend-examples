package slogctx

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
)

type MultilineTextHandler struct {
	writer io.Writer
	mu     *sync.Mutex
	attrs  []slog.Attr
	groups []string
}

func NewMultilineTextHandler(writer io.Writer) *MultilineTextHandler {
	return &MultilineTextHandler{writer: writer, mu: &sync.Mutex{}}
}

func (h *MultilineTextHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *MultilineTextHandler) Handle(_ context.Context, record slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	attrs := []slog.Attr{
		slog.Time("time", record.Time),
		slog.String("level", record.Level.String()),
		slog.String("msg", record.Message),
	}
	attrs = append(attrs, h.attrs...)
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})

	for _, attr := range flattenAttrs(h.groups, attrs) {
		err := h.writeAttr(attr)
		if err != nil {
			return err
		}
	}

	_, err := fmt.Fprintln(h.writer)
	if err != nil {
		return fmt.Errorf("write log record separator: %w", err)
	}

	return nil
}

func (h *MultilineTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	childAttrs := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	childAttrs = append(childAttrs, h.attrs...)
	childAttrs = append(childAttrs, attrs...)

	return &MultilineTextHandler{
		writer: h.writer,
		mu:     h.mu,
		attrs:  childAttrs,
		groups: h.groups,
	}
}

func (h *MultilineTextHandler) WithGroup(name string) slog.Handler {
	groups := make([]string, 0, len(h.groups)+1)
	groups = append(groups, h.groups...)
	groups = append(groups, name)

	return &MultilineTextHandler{
		writer: h.writer,
		mu:     h.mu,
		attrs:  h.attrs,
		groups: groups,
	}
}

func (h *MultilineTextHandler) writeAttr(attr slog.Attr) error {
	_, err := fmt.Fprintf(h.writer, "%s=%s\n", attr.Key, formatValue(attr.Value))
	if err != nil {
		return fmt.Errorf("write log attr: %w", err)
	}

	return nil
}

func flattenAttrs(groups []string, attrs []slog.Attr) []slog.Attr {
	flattened := make([]slog.Attr, 0, len(attrs))
	for _, attr := range attrs {
		flattened = append(flattened, flattenAttr(groups, attr)...)
	}

	return flattened
}

func flattenAttr(groups []string, attr slog.Attr) []slog.Attr {
	attr.Value = attr.Value.Resolve()
	key := strings.Join(append(groups, attr.Key), ".")
	if attr.Value.Kind() != slog.KindGroup {
		return []slog.Attr{slog.Any(key, attr.Value.Any())}
	}

	attrs := attr.Value.Group()
	flattened := make([]slog.Attr, 0, len(attrs))
	for _, child := range attrs {
		flattened = append(flattened, flattenAttr(append(groups, attr.Key), child)...)
	}

	return flattened
}

func formatValue(value slog.Value) string {
	value = value.Resolve()

	switch value.Kind() {
	case slog.KindString:
		return formatString(value.String())
	case slog.KindTime:
		return value.Time().Format(time.RFC3339Nano)
	case slog.KindDuration:
		return value.Duration().String()
	case slog.KindBool:
		return strconv.FormatBool(value.Bool())
	case slog.KindInt64:
		return strconv.FormatInt(value.Int64(), 10)
	case slog.KindUint64:
		return strconv.FormatUint(value.Uint64(), 10)
	case slog.KindFloat64:
		return strconv.FormatFloat(value.Float64(), 'g', -1, 64)
	case slog.KindAny, slog.KindGroup, slog.KindLogValuer:
		return fmt.Sprint(value.Any())
	default:
		return fmt.Sprint(value.Any())
	}
}

func formatString(value string) string {
	if value == "" || strings.ContainsAny(value, " \t\n\r\"=") {
		return strconv.Quote(value)
	}

	return value
}
