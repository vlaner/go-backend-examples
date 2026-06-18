package userservice

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/vlaner/go-backend-examples/logging/domain"
)

type CreateUser interface {
	CreateUser(ctx context.Context, command CreateUserCommand) (domain.User, error)
}

type CreateUserCommand struct {
	Username string
	Password string
}

func (c CreateUserCommand) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("username", c.Username),
		slog.String("password", "[REDACTED]"),
	)
}

type Service struct {
	users  domain.UserRepository
	logger *slog.Logger
}

func New(users domain.UserRepository, logger *slog.Logger) *Service {
	return &Service{
		users:  users,
		logger: logger.With(slog.String("component", "user.service")),
	}
}

func (s *Service) CreateUser(ctx context.Context, command CreateUserCommand) (domain.User, error) {
	user := domain.User{ID: uuid.New(), Username: command.Username}

	s.logger.InfoContext(ctx, "creating user", slog.String("user_id", user.ID.String()), slog.String("username", user.Username))

	user, err := s.users.Create(ctx, user, command.Password)
	if err != nil {
		return domain.User{}, fmt.Errorf("create user: %w", err)
	}

	_, err = s.users.CountByUsername(ctx, user.Username)
	if err != nil {
		return domain.User{}, fmt.Errorf("count users by username: %w", err)
	}

	s.logger.InfoContext(ctx, "created user", slog.String("user_id", user.ID.String()), slog.String("username", user.Username))

	return user, nil
}
