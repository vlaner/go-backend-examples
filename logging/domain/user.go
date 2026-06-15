package domain

import (
	"context"

	"github.com/google/uuid"
)

type User struct {
	ID       uuid.UUID
	Username string
}

type UserRepository interface {
	Create(ctx context.Context, user User, password string) (User, error)
}
