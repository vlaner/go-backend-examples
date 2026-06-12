package unitofwork

import "context"

type UnitOfWork[T any] interface {
	Do(ctx context.Context, fn func(ctx context.Context, value T) error) error
}
