package visitor

type Step struct {
	Depth       int
	Rule        Rule
	Amount      Money
	Contributed bool
}

func Trace(rule Rule, cart Cart) ([]Step, error) {
	return Walk(rule, tracer{cart: cart})
}

type tracer struct {
	cart  Cart
	depth int
}

func (t tracer) child() tracer { return tracer{cart: t.cart, depth: t.depth + 1} }

func (t tracer) Percent(n Percent) ([]Step, error) {
	amount := t.cart.Subtotal * Money(n.Pct) / centsPerUnit
	return []Step{{Depth: t.depth, Rule: n, Amount: amount, Contributed: true}}, nil
}

func (t tracer) Flat(n Flat) ([]Step, error) {
	return []Step{{Depth: t.depth, Rule: n, Amount: n.Amount, Contributed: true}}, nil
}

func (t tracer) MinSpend(n MinSpend) ([]Step, error) {
	if t.cart.Subtotal < n.Threshold {
		return []Step{{Depth: t.depth, Rule: n}}, nil
	}
	return Walk(n.Discount, t.child())
}

func (t tracer) All(n All) ([]Step, error) {
	steps := make([]Step, 0, len(n.Rules))
	for _, rule := range n.Rules {
		child, err := Walk(rule, t.child())
		if err != nil {
			return nil, err
		}
		steps = append(steps, child...)
	}
	return steps, nil
}

func (t tracer) Best(n Best) ([]Step, error) {
	var steps []Step
	winnerStart, winnerEnd := 0, 0
	bestAmount := Money(-1)
	for _, rule := range n.Rules {
		start := len(steps)
		child, err := Walk(rule, t.child())
		if err != nil {
			return nil, err
		}
		steps = append(steps, child...)
		if amount := contributed(child); amount > bestAmount {
			winnerStart, winnerEnd, bestAmount = start, len(steps), amount
		}
	}
	for i := range steps {
		if i < winnerStart || i >= winnerEnd {
			steps[i].Contributed = false
		}
	}
	return steps, nil
}

func contributed(steps []Step) Money {
	var total Money
	for _, step := range steps {
		if step.Contributed {
			total += step.Amount
		}
	}
	return total
}
