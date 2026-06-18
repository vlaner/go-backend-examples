package optional

type Optional[T any] struct {
	value T
	set   bool
	null  bool
}

func Null[T any]() Optional[T] {
	return Optional[T]{set: true, null: true}
}

func Value[T any](value T) Optional[T] {
	return Optional[T]{set: true, value: value}
}

func (n Optional[T]) IsSet() bool {
	return n.set
}

func (n Optional[T]) IsNull() bool {
	return n.set && n.null
}

func (n Optional[T]) Value() (T, bool) {
	if !n.set || n.null {
		var zero T
		return zero, false
	}

	return n.value, true
}

func (n Optional[T]) Ptr() *T {
	if !n.set || n.null {
		return nil
	}

	return &n.value
}
