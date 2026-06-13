package approach3

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vlaner/go-backend-examples/pgx-transactions/profilerepo"
	"github.com/vlaner/go-backend-examples/pgx-transactions/userrepo"
)

type Application struct {
	UserService UserService
}

func NewApplication(userService UserService) *Application {
	return &Application{UserService: userService}
}

func (a *Application) CreateUserWithProfile(ctx context.Context, input CreateUserWithProfileInput) (CreateUserWithProfileResult, error) {
	result, err := a.UserService.CreateUserWithProfile(ctx, input)
	if err != nil {
		return CreateUserWithProfileResult{}, fmt.Errorf("create user with profile: %w", err)
	}

	return result, nil
}

func NewPGXTransactionalUserService(pool *pgxpool.Pool) UserService {
	return pgxTransactionalUserService{pool: pool}
}

type pgxTransactionalUserService struct {
	pool *pgxpool.Pool
}

func (s pgxTransactionalUserService) CreateUserWithProfile(ctx context.Context, input CreateUserWithProfileInput) (CreateUserWithProfileResult, error) {
	var result CreateUserWithProfileResult
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		service := NewUserService(
			userrepo.New(tx),
			profilerepo.New(tx),
		)

		var err error
		result, err = service.CreateUserWithProfile(ctx, input)
		if err != nil {
			return fmt.Errorf("create user with profile: %w", err)
		}

		return nil
	})
	if err != nil {
		return CreateUserWithProfileResult{}, fmt.Errorf("transactional user service: %w", err)
	}

	return result, nil
}
