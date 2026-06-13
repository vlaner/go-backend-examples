package approach4

import (
	"context"
	"fmt"

	"github.com/vlaner/go-backend-examples/pgx-transactions/unitofwork"
)

type transactionalCompleteOrderUseCase struct {
	uow unitofwork.UnitOfWork[CompleteOrderUseCase]
}

func NewTransactionalCompleteOrderUseCase(
	uow unitofwork.UnitOfWork[CompleteOrderUseCase],
) CompleteOrderUseCase {
	return &transactionalCompleteOrderUseCase{uow: uow}
}

func (u *transactionalCompleteOrderUseCase) Complete(ctx context.Context, cmd CompleteOrderCommand) error {
	err := u.uow.Do(ctx, func(ctx context.Context, useCase CompleteOrderUseCase) error {
		err := useCase.Complete(ctx, cmd)
		if err != nil {
			return fmt.Errorf("complete order: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("transactional complete order: %w", err)
	}

	return nil
}
