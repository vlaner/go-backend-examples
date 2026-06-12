package approach2

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/vlaner/go-backend-examples/pgx-transactions/domain"
	"github.com/vlaner/go-backend-examples/pgx-transactions/unitofwork"
)

type UserService struct {
	uow unitofwork.UnitOfWork[Repositories]
}

type CreateUserWithProfileInput struct {
	Username         string
	Password         string
	Description      string
	Contact          string
	SocialMediaLinks []string
}

type CreateUserWithProfileResult struct {
	User    domain.User
	Profile domain.Profile
}

func NewUserService(uow unitofwork.UnitOfWork[Repositories]) *UserService {
	return &UserService{uow: uow}
}

func (s *UserService) CreateUserWithProfile(ctx context.Context, input CreateUserWithProfileInput) (CreateUserWithProfileResult, error) {
	user := domain.User{
		ID:       uuid.New(),
		Username: input.Username,
		Password: input.Password,
	}
	profile := domain.Profile{
		UserID:           user.ID,
		Description:      input.Description,
		Contact:          input.Contact,
		SocialMediaLinks: input.SocialMediaLinks,
	}

	err := s.uow.Do(ctx, func(ctx context.Context, repos Repositories) error {
		err := repos.Users().Create(ctx, user)
		if err != nil {
			return fmt.Errorf("create user: %w", err)
		}

		err = repos.Profiles().Create(ctx, profile)
		if err != nil {
			return fmt.Errorf("create profile: %w", err)
		}

		return nil
	})
	if err != nil {
		return CreateUserWithProfileResult{}, fmt.Errorf("create user with profile: %w", err)
	}

	return CreateUserWithProfileResult{User: user, Profile: profile}, nil
}
