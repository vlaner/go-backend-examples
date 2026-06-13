package domain

import (
	"context"
	"errors"
)

var ErrInvalidOrderState = errors.New("invalid order state")

type Order struct {
	ID     string
	Status OrderStatus
}

type OrderStatus string

const (
	OrderStatusPendingPayment OrderStatus = "pending_payment"
	OrderStatusPaid           OrderStatus = "paid"
)

type CreatePaymentInput struct {
	OrderID           string
	ProviderPaymentID string
	Amount            int64
}

type OrderRepository interface {
	GetByID(ctx context.Context, orderID string) (Order, error)
	MarkPaid(ctx context.Context, orderID string) error
}

type InventoryRepository interface {
	ReserveForOrder(ctx context.Context, orderID string) error
}

type PaymentRepository interface {
	Create(ctx context.Context, input CreatePaymentInput) error
}
