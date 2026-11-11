package visitor

import (
	"errors"
	"fmt"
	"strings"
)

const (
	minPercent = 1
	maxPercent = 100
)

var ErrInvalidRule = errors.New("invalid rule")

func Problems(rule Rule) ([]string, error) {
	v := &validator{}
	_, err := Walk(rule, v)
	if err != nil {
		return nil, err
	}
	return v.problems, nil
}

func Validate(rule Rule) error {
	problems, err := Problems(rule)
	if err != nil {
		return err
	}
	if len(problems) > 0 {
		return fmt.Errorf("%w: %s", ErrInvalidRule, strings.Join(problems, "; "))
	}
	return nil
}

type validator struct{ problems []string }

func (v *validator) Percent(n Percent) ([]string, error) {
	if n.Pct < minPercent || n.Pct > maxPercent {
		v.problems = append(v.problems, fmt.Sprintf("percent must be between %d and %d, got %d", minPercent, maxPercent, n.Pct))
	}
	return v.problems, nil
}

func (v *validator) Flat(n Flat) ([]string, error) {
	if n.Amount <= 0 {
		v.problems = append(v.problems, fmt.Sprintf("flat must be positive, got %d", n.Amount))
	}
	return v.problems, nil
}

func (v *validator) MinSpend(n MinSpend) ([]string, error) {
	if n.Threshold <= 0 {
		v.problems = append(v.problems, fmt.Sprintf("min_spend must be positive, got %d", n.Threshold))
	}
	_, err := Walk(n.Discount, v)
	if err != nil {
		return nil, err
	}
	return v.problems, nil
}

func (v *validator) All(n All) ([]string, error) {
	if len(n.Rules) == 0 {
		v.problems = append(v.problems, "all must contain at least one rule")
	}
	for _, rule := range n.Rules {
		_, err := Walk(rule, v)
		if err != nil {
			return nil, err
		}
	}
	return v.problems, nil
}

func (v *validator) Best(n Best) ([]string, error) {
	if len(n.Rules) == 0 {
		v.problems = append(v.problems, "best must contain at least one rule")
	}
	for _, rule := range n.Rules {
		_, err := Walk(rule, v)
		if err != nil {
			return nil, err
		}
	}
	return v.problems, nil
}
