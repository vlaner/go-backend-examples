package approach3

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/vlaner/go-backend-examples/pgx-transactions/domain"
)

type UserService interface {
	CreateUserWithProfile(ctx context.Context, input CreateUserWithProfileInput) (CreateUserWithProfileResult, error)
}

type userService struct {
	users    domain.UserRepository
	profiles domain.ProfileRepository
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

func NewUserService(users domain.UserRepository, profiles domain.ProfileRepository) UserService {
	return &userService{
		users:    users,
		profiles: profiles,
	}
}

func (s *userService) CreateUserWithProfile(ctx context.Context, input CreateUserWithProfileInput) (CreateUserWithProfileResult, error) {
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

	err := s.users.Create(ctx, user)
	if err != nil {
		return CreateUserWithProfileResult{}, fmt.Errorf("create user: %w", err)
	}

	err = s.profiles.Create(ctx, profile)
	if err != nil {
		return CreateUserWithProfileResult{}, fmt.Errorf("create profile: %w", err)
	}

	return CreateUserWithProfileResult{User: user, Profile: profile}, nil
}
