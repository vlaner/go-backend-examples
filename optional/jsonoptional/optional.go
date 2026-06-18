package jsonoptional

import (
	"bytes"
	"encoding/json"
	"fmt"

	optional "github.com/vlaner/go-backend-examples/optional/optional"
)

type Optional[T any] struct {
	value optional.Optional[T]
}

func (o Optional[T]) Optional() optional.Optional[T] {
	return o.value
}

func (o Optional[T]) IsZero() bool {
	return !o.value.IsSet()
}

func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		o.value = optional.Null[T]()
		return nil
	}

	var value T
	err := json.Unmarshal(data, &value)
	if err != nil {
		return fmt.Errorf("unmarshal optional: %w", err)
	}

	o.value = optional.Value(value)
	return nil
}

func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if !o.value.IsSet() || o.value.IsNull() {
		return []byte("null"), nil
	}

	value, _ := o.value.Value()
	b, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal optional: %w", err)
	}

	return b, nil
}
