package approach3

import (
	"context"
	"fmt"

	"github.com/vlaner/go-backend-examples/pgx-transactions/unitofwork"
)

type unitOfWorkUserService struct {
	uow unitofwork.UnitOfWork[UserService]
}

func NewUnitOfWorkUserService(uow unitofwork.UnitOfWork[UserService]) UserService {
	return &unitOfWorkUserService{uow: uow}
}

func (s *unitOfWorkUserService) CreateUserWithProfile(
	ctx context.Context,
	input CreateUserWithProfileInput,
) (CreateUserWithProfileResult, error) {
	var result CreateUserWithProfileResult
	err := s.uow.Do(ctx, func(ctx context.Context, service UserService) error {
		var err error
		result, err = service.CreateUserWithProfile(ctx, input)
		if err != nil {
			return fmt.Errorf("create user with profile: %w", err)
		}

		return nil
	})
	if err != nil {
		return CreateUserWithProfileResult{}, fmt.Errorf("unit of work create user with profile: %w", err)
	}

	return result, nil
}
