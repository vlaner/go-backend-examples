package visitor_test

import (
	"errors"
	"reflect"
	"testing"

	visitor "github.com/vlaner/go-backend-examples/enums/visitor"
)

func TestApply(t *testing.T) {
	t.Parallel()
	cart := visitor.Cart{Subtotal: 3000}
	tests := []struct {
		name string
		rule visitor.Rule
		want visitor.Money
	}{
		{"percent", visitor.Percent{Pct: 10}, 300},
		{"percent_rounds_down", visitor.Percent{Pct: 33}, 990},
		{"flat", visitor.Flat{Amount: 200}, 200},
		{"flat_capped_at_subtotal", visitor.Flat{Amount: 5000}, 3000},
		{"min_spend_below_threshold", visitor.MinSpend{Threshold: 5000, Discount: visitor.Flat{Amount: 500}}, 0},
		{"min_spend_met", visitor.MinSpend{Threshold: 2500, Discount: visitor.Flat{Amount: 500}}, 500},
		{"all_sums_children", visitor.All{Rules: []visitor.Rule{visitor.Percent{Pct: 10}, visitor.Flat{Amount: 200}}}, 500},
		{"all_capped_at_subtotal", visitor.All{Rules: []visitor.Rule{visitor.Flat{Amount: 2000}, visitor.Flat{Amount: 2000}}}, 3000},
		{"best_takes_highest", visitor.Best{Rules: []visitor.Rule{visitor.Flat{Amount: 200}, visitor.Percent{Pct: 15}}}, 450},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := visitor.Apply(tc.rule, cart)
			if err != nil {
				t.Fatalf("Apply: %v", err)
			}
			if got != tc.want {
				t.Fatalf("Apply = %d cents, want %d", got, tc.want)
			}
		})
	}
}

func TestTrace(t *testing.T) {
	t.Parallel()
	cart := visitor.Cart{Subtotal: 3000}
	tests := []struct {
		name string
		rule visitor.Rule
		want []visitor.Step
	}{
		{
			name: "all_reports_skipped_min_spend",
			rule: visitor.All{Rules: []visitor.Rule{
				visitor.Percent{Pct: 10},
				visitor.MinSpend{Threshold: 5000, Discount: visitor.Flat{Amount: 500}},
				visitor.Flat{Amount: 200},
			}},
			want: []visitor.Step{
				{Depth: 1, Rule: visitor.Percent{Pct: 10}, Amount: 300, Contributed: true},
				{Depth: 1, Rule: visitor.MinSpend{Threshold: 5000, Discount: visitor.Flat{Amount: 500}}, Amount: 0, Contributed: false},
				{Depth: 1, Rule: visitor.Flat{Amount: 200}, Amount: 200, Contributed: true},
			},
		},
		{
			name: "min_spend_met_reports_inner_rule",
			rule: visitor.MinSpend{Threshold: 2500, Discount: visitor.Flat{Amount: 500}},
			want: []visitor.Step{
				{Depth: 1, Rule: visitor.Flat{Amount: 500}, Amount: 500, Contributed: true},
			},
		},
		{
			name: "best_marks_every_candidate_but_the_winner",
			rule: visitor.Best{Rules: []visitor.Rule{visitor.Flat{Amount: 200}, visitor.Percent{Pct: 15}}},
			want: []visitor.Step{
				{Depth: 1, Rule: visitor.Flat{Amount: 200}, Amount: 200, Contributed: false},
				{Depth: 1, Rule: visitor.Percent{Pct: 15}, Amount: 450, Contributed: true},
			},
		},
		{
			name: "nested_groups_increase_depth",
			rule: visitor.All{Rules: []visitor.Rule{visitor.Best{Rules: []visitor.Rule{visitor.Percent{Pct: 10}}}}},
			want: []visitor.Step{
				{Depth: 2, Rule: visitor.Percent{Pct: 10}, Amount: 300, Contributed: true},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := visitor.Trace(tc.rule, cart)
			if err != nil {
				t.Fatalf("Trace: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Trace:\nwant %#v\ngot  %#v", tc.want, got)
			}
		})
	}
}

func TestUnknownRuleIsReported(t *testing.T) {
	t.Parallel()
	cart := visitor.Cart{Subtotal: 3000}
	_, err := visitor.Apply(nil, cart)
	if !errors.Is(err, visitor.ErrUnknownRule) {
		t.Fatalf("Apply error = %v, want ErrUnknownRule", err)
	}
	_, err = visitor.Trace(visitor.All{Rules: []visitor.Rule{nil}}, cart)
	if !errors.Is(err, visitor.ErrUnknownRule) {
		t.Fatalf("nested Trace error = %v, want ErrUnknownRule", err)
	}
	_, err = visitor.Problems(visitor.Best{Rules: []visitor.Rule{nil}})
	if !errors.Is(err, visitor.ErrUnknownRule) {
		t.Fatalf("nested Problems error = %v, want ErrUnknownRule", err)
	}
	err = visitor.Validate(nil)
	if !errors.Is(err, visitor.ErrUnknownRule) {
		t.Fatalf("Validate error = %v, want ErrUnknownRule", err)
	}
}

func TestTraceExplainsApply(t *testing.T) {
	t.Parallel()
	cart := visitor.Cart{Subtotal: 3000}
	rules := []visitor.Rule{
		visitor.Percent{Pct: 33},
		visitor.All{Rules: []visitor.Rule{visitor.Percent{Pct: 10}, visitor.Flat{Amount: 200}}},
		visitor.Best{Rules: []visitor.Rule{visitor.Flat{Amount: 200}, visitor.Percent{Pct: 15}}},
		visitor.Best{Rules: []visitor.Rule{visitor.MinSpend{Threshold: 5000, Discount: visitor.Flat{Amount: 900}}, visitor.Flat{Amount: 100}}},
		visitor.All{Rules: []visitor.Rule{
			visitor.Best{Rules: []visitor.Rule{visitor.Percent{Pct: 5}, visitor.Flat{Amount: 50}}},
			visitor.MinSpend{Threshold: 2500, Discount: visitor.Flat{Amount: 75}},
		}},
	}
	for _, rule := range rules {
		applied, err := visitor.Apply(rule, cart)
		if err != nil {
			t.Fatalf("Apply: %v", err)
		}
		steps, err := visitor.Trace(rule, cart)
		if err != nil {
			t.Fatalf("Trace: %v", err)
		}
		var contributed visitor.Money
		for _, step := range steps {
			if step.Contributed {
				contributed += step.Amount
			}
		}
		if applied != contributed {
			t.Fatalf("trace of %#v sums to %d cents, Apply says %d", rule, contributed, applied)
		}
	}
}

func TestValidateAcceptsEveryRuleType(t *testing.T) {
	t.Parallel()
	valid := visitor.All{Rules: []visitor.Rule{
		visitor.Percent{Pct: 100},
		visitor.MinSpend{Threshold: 2500, Discount: visitor.Flat{Amount: 500}},
		visitor.Best{Rules: []visitor.Rule{visitor.Flat{Amount: 1}}},
	}}
	err := visitor.Validate(valid)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	problems, err := visitor.Problems(valid)
	if err != nil {
		t.Fatalf("Problems: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("Problems = %q, want none", problems)
	}
}

func TestValidateRejectsInvalidRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		rule visitor.Rule
	}{
		{"percent_zero", visitor.Percent{Pct: 0}},
		{"percent_over_100", visitor.Percent{Pct: 101}},
		{"percent_negative", visitor.Percent{Pct: -5}},
		{"flat_zero", visitor.Flat{Amount: 0}},
		{"min_spend_zero", visitor.MinSpend{Threshold: 0, Discount: visitor.Flat{Amount: 100}}},
		{"empty_all", visitor.All{}},
		{"empty_best", visitor.Best{}},
		{"nested_problem", visitor.All{Rules: []visitor.Rule{visitor.Flat{Amount: 100}, visitor.MinSpend{Threshold: 0, Discount: visitor.Percent{Pct: 0}}}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := visitor.Validate(tc.rule)
			if !errors.Is(err, visitor.ErrInvalidRule) {
				t.Fatalf("Validate error = %v, want ErrInvalidRule", err)
			}
		})
	}
}

func TestProblemsReportsEveryProblem(t *testing.T) {
	t.Parallel()
	nested := visitor.All{Rules: []visitor.Rule{visitor.Flat{Amount: 100}, visitor.MinSpend{Threshold: 0, Discount: visitor.Percent{Pct: 0}}}}
	problems, err := visitor.Problems(nested)
	if err != nil {
		t.Fatalf("Problems: %v", err)
	}
	if got, want := len(problems), 2; got != want {
		t.Fatalf("Problems reported %d problems, want %d", got, want)
	}
	problems, err = visitor.Problems(visitor.All{})
	if err != nil {
		t.Fatalf("Problems: %v", err)
	}
	if got, want := len(problems), 1; got != want {
		t.Fatalf("Problems on empty All reported %d problems, want %d", got, want)
	}
}

func TestApplyNeverExceedsSubtotal(t *testing.T) {
	t.Parallel()
	rule := visitor.Best{Rules: []visitor.Rule{visitor.Flat{Amount: 1000}, visitor.Flat{Amount: 2000}}}
	free := visitor.Cart{Subtotal: 500}
	got, err := visitor.Apply(rule, free)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got != free.Subtotal {
		t.Fatalf("Apply = %d cents, want cart subtotal %d", got, free.Subtotal)
	}
}
