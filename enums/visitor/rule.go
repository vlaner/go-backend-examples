package visitor

import (
	"errors"
	"fmt"
)

const centsPerUnit = 100

var ErrUnknownRule = errors.New("unknown rule")

type Money int

type Cart struct{ Subtotal Money }

type Rule interface{ isRule() }

type (
	Percent  struct{ Pct int }
	Flat     struct{ Amount Money }
	MinSpend struct {
		Threshold Money
		Discount  Rule
	}
	All  struct{ Rules []Rule }
	Best struct{ Rules []Rule }
)

func (Percent) isRule()  {}
func (Flat) isRule()     {}
func (MinSpend) isRule() {}
func (All) isRule()      {}
func (Best) isRule()     {}

type Visitor[T any] interface {
	Percent(Percent) (T, error)
	Flat(Flat) (T, error)
	MinSpend(MinSpend) (T, error)
	All(All) (T, error)
	Best(Best) (T, error)
}

func Walk[T any](rule Rule, v Visitor[T]) (T, error) {
	var value T
	var err error
	switch node := rule.(type) {
	case Percent:
		value, err = v.Percent(node)
	case Flat:
		value, err = v.Flat(node)
	case MinSpend:
		value, err = v.MinSpend(node)
	case All:
		value, err = v.All(node)
	case Best:
		value, err = v.Best(node)
	default:
		return value, fmt.Errorf("%w: %T", ErrUnknownRule, rule)
	}
	if err != nil {
		return value, fmt.Errorf("walk %T: %w", rule, err)
	}

	return value, nil
}
