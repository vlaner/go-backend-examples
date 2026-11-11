package visitor

func Apply(rule Rule, cart Cart) (Money, error) {
	discount, err := Walk(rule, apply{cart: cart})
	if err != nil {
		return 0, err
	}
	return clampDiscount(discount, cart.Subtotal), nil
}

func clampDiscount(discount, subtotal Money) Money {
	if discount < 0 {
		return 0
	}
	if discount > subtotal {
		return subtotal
	}
	return discount
}

type apply struct{ cart Cart }

func (a apply) Percent(n Percent) (Money, error) {
	return a.cart.Subtotal * Money(n.Pct) / centsPerUnit, nil
}

func (apply) Flat(n Flat) (Money, error) { return n.Amount, nil }

func (a apply) MinSpend(n MinSpend) (Money, error) {
	if a.cart.Subtotal < n.Threshold {
		return 0, nil
	}
	return Walk(n.Discount, a)
}

func (a apply) All(n All) (Money, error) {
	var total Money
	for _, rule := range n.Rules {
		amount, err := Walk(rule, a)
		if err != nil {
			return 0, err
		}
		total += amount
	}
	return total, nil
}

func (a apply) Best(n Best) (Money, error) {
	var best Money
	for _, rule := range n.Rules {
		amount, err := Walk(rule, a)
		if err != nil {
			return 0, err
		}
		best = max(best, amount)
	}
	return best, nil
}
