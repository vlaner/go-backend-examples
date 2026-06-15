package approachtransactionalservice

import (
	"context"
	"fmt"
)

type TxManager interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type Transactions struct {
	CreateUserWithProfile TxManager
}

type txManagerUserService struct {
	next UserService
	tx   Transactions
}

func NewTxManagerUserService(next UserService, tx Transactions) UserService {
	return &txManagerUserService{next: next, tx: tx}
}

func (s *txManagerUserService) CreateUserWithProfile(
	ctx context.Context,
	input CreateUserWithProfileInput,
) (CreateUserWithProfileResult, error) {
	var result CreateUserWithProfileResult
	err := s.tx.CreateUserWithProfile.Do(ctx, func(ctx context.Context) error {
		var err error
		result, err = s.next.CreateUserWithProfile(ctx, input)
		if err != nil {
			return fmt.Errorf("create user with profile: %w", err)
		}

		return nil
	})
	if err != nil {
		return CreateUserWithProfileResult{}, fmt.Errorf("tx manager create user with profile: %w", err)
	}

	return result, nil
}
