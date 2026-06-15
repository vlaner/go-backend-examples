package approachusecaseboundary

import (
	"context"
	"fmt"

	"github.com/vlaner/go-backend-examples/pgx-transactions/domain"
)

type CompleteOrderUseCase interface {
	Complete(ctx context.Context, cmd CompleteOrderCommand) error
}

type CompleteOrderCommand struct {
	OrderID   string
	PaymentID string
	Amount    int64
}

type completeOrderUseCase struct {
	orders    domain.OrderRepository
	inventory domain.InventoryRepository
	payments  domain.PaymentRepository
}

func NewCompleteOrderUseCase(
	orders domain.OrderRepository,
	inventory domain.InventoryRepository,
	payments domain.PaymentRepository,
) CompleteOrderUseCase {
	return &completeOrderUseCase{
		orders:    orders,
		inventory: inventory,
		payments:  payments,
	}
}

func (s *completeOrderUseCase) Complete(ctx context.Context, cmd CompleteOrderCommand) error {
	order, err := s.orders.GetByID(ctx, cmd.OrderID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}

	if order.Status == domain.OrderStatusPaid {
		return nil
	}

	if order.Status != domain.OrderStatusPendingPayment {
		return domain.ErrInvalidOrderState
	}

	err = s.inventory.ReserveForOrder(ctx, order.ID)
	if err != nil {
		return fmt.Errorf("reserve inventory: %w", err)
	}

	err = s.payments.Create(ctx, domain.CreatePaymentInput{
		OrderID:           order.ID,
		ProviderPaymentID: cmd.PaymentID,
		Amount:            cmd.Amount,
	})
	if err != nil {
		return fmt.Errorf("create payment: %w", err)
	}

	err = s.orders.MarkPaid(ctx, order.ID)
	if err != nil {
		return fmt.Errorf("mark order paid: %w", err)
	}

	return nil
}
