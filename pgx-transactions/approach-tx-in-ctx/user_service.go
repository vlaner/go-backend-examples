package approachtxinctx

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/vlaner/go-backend-examples/pgx-transactions/domain"
)

type TxManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type UserService struct {
	txManager TxManager
	users     domain.UserRepository
	profiles  domain.ProfileRepository
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

func NewUserService(txManager TxManager, users domain.UserRepository, profiles domain.ProfileRepository) *UserService {
	return &UserService{
		txManager: txManager,
		users:     users,
		profiles:  profiles,
	}
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

	err := s.txManager.WithTx(ctx, func(ctx context.Context) error {
		err := s.users.Create(ctx, user)
		if err != nil {
			return fmt.Errorf("create user: %w", err)
		}

		err = s.profiles.Create(ctx, profile)
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
