package userservice

import (
	"context"
	"fmt"

	"github.com/vlaner/go-backend-examples/logging/canonicallog"
	"github.com/vlaner/go-backend-examples/logging/domain"
)

type CanonicalCreateUser struct {
	next CreateUser
}

func NewCanonicalCreateUser(next CreateUser) *CanonicalCreateUser {
	return &CanonicalCreateUser{next: next}
}

func (c *CanonicalCreateUser) CreateUser(ctx context.Context, command CreateUserCommand) (domain.User, error) {
	canonicallog.Set(ctx, "service.operation", "create_user")
	canonicallog.Set(ctx, "service.command", command)

	user, err := c.next.CreateUser(ctx, command)
	if err != nil {
		canonicallog.Set(ctx, "service.outcome", "failed")
		canonicallog.Set(ctx, "service.error", err)
		return domain.User{}, fmt.Errorf("canonical create user: %w", err)
	}

	canonicallog.Set(ctx, "service.outcome", "created")
	canonicallog.Set(ctx, "service.user_id", user.ID.String())

	return user, nil
}
