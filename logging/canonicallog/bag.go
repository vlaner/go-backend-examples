package canonicallog

import (
	"context"
	"log/slog"
	"sort"
	"strconv"
	"sync"
	"time"
)

type contextKey struct{}

type Bag struct {
	mu     sync.Mutex
	fields map[string]any
}

type groupRecord struct {
	attrs []slog.Attr
}

func NewBag() *Bag {
	return &Bag{fields: make(map[string]any)}
}

func WithBag(ctx context.Context, bag *Bag) context.Context {
	return context.WithValue(ctx, contextKey{}, bag)
}

func Set(ctx context.Context, key string, value any) {
	bag, ok := ctx.Value(contextKey{}).(*Bag)
	if !ok {
		return
	}

	bag.mu.Lock()
	defer bag.mu.Unlock()

	bag.fields[key] = value
}

func Add(ctx context.Context, key string, delta int64) {
	bag, ok := ctx.Value(contextKey{}).(*Bag)
	if !ok {
		return
	}

	bag.mu.Lock()
	defer bag.mu.Unlock()

	current, _ := bag.fields[key].(int64)
	bag.fields[key] = current + delta
}

func AddDuration(ctx context.Context, key string, duration time.Duration) {
	bag, ok := ctx.Value(contextKey{}).(*Bag)
	if !ok {
		return
	}

	bag.mu.Lock()
	defer bag.mu.Unlock()

	current, _ := bag.fields[key].(time.Duration)
	bag.fields[key] = current + duration
}

func Append(ctx context.Context, key string, value any) {
	bag, ok := ctx.Value(contextKey{}).(*Bag)
	if !ok {
		return
	}

	bag.mu.Lock()
	defer bag.mu.Unlock()

	values, _ := bag.fields[key].([]any)
	bag.fields[key] = append(values, value)
}

func AppendGroup(ctx context.Context, key string, attrs ...slog.Attr) {
	bag, ok := ctx.Value(contextKey{}).(*Bag)
	if !ok {
		return
	}

	bag.mu.Lock()
	defer bag.mu.Unlock()

	records, _ := bag.fields[key].([]groupRecord)
	bag.fields[key] = append(records, groupRecord{attrs: attrs})
}

func Attrs(ctx context.Context) []any {
	bag, ok := ctx.Value(contextKey{}).(*Bag)
	if !ok {
		return nil
	}

	bag.mu.Lock()
	defer bag.mu.Unlock()

	keys := make([]string, 0, len(bag.fields))
	for key := range bag.fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	attrs := make([]any, 0, len(keys))
	for _, key := range keys {
		attrs = append(attrs, attr(key, bag.fields[key]))
	}

	return attrs
}

func attr(key string, value any) any {
	records, ok := value.([]groupRecord)
	if !ok {
		return slog.Any(key, value)
	}

	groups := make([]any, 0, len(records))
	for i, record := range records {
		groups = append(groups, slog.Group(strconv.Itoa(i), attrsToAny(record.attrs)...))
	}

	return slog.Group(key, groups...)
}

func attrsToAny(attrs []slog.Attr) []any {
	values := make([]any, len(attrs))
	for i, attr := range attrs {
		values[i] = attr
	}

	return values
}
