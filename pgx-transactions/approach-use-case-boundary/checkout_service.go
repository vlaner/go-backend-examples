package approachusecaseboundary

import (
	"context"
	"fmt"
)

type PricingService interface {
	GetQuote(ctx context.Context, cartID string) (Quote, error)
}

type PaymentGateway interface {
	Authorize(ctx context.Context, cmd AuthorizePaymentCommand) (PaymentAuthorization, error)
}

type CheckoutService struct {
	pricing       PricingService
	payment       PaymentGateway
	completeOrder CompleteOrderUseCase
}

type CheckoutCommand struct {
	CartID  string
	OrderID string
}

type Quote struct {
	Amount int64
}

type AuthorizePaymentCommand struct {
	OrderID string
	Amount  int64
}

type PaymentAuthorization struct {
	ID string
}

func NewCheckoutService(
	pricing PricingService,
	payment PaymentGateway,
	completeOrder CompleteOrderUseCase,
) *CheckoutService {
	return &CheckoutService{
		pricing:       pricing,
		payment:       payment,
		completeOrder: completeOrder,
	}
}

func (s *CheckoutService) Checkout(ctx context.Context, cmd CheckoutCommand) error {
	// No transaction: quote calculation may call external pricing services.
	quote, err := s.pricing.GetQuote(ctx, cmd.CartID)
	if err != nil {
		return fmt.Errorf("get quote: %w", err)
	}

	// No transaction: payment authorization calls the payment provider.
	payment, err := s.payment.Authorize(ctx, AuthorizePaymentCommand{
		OrderID: cmd.OrderID,
		Amount:  quote.Amount,
	})
	if err != nil {
		return fmt.Errorf("authorize payment: %w", err)
	}

	// One transaction: the complete order command is the transaction boundary.
	err = s.completeOrder.Complete(ctx, CompleteOrderCommand{
		OrderID:   cmd.OrderID,
		PaymentID: payment.ID,
		Amount:    quote.Amount,
	})
	if err != nil {
		return fmt.Errorf("complete order: %w", err)
	}

	return nil
}
