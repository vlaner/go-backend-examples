package domain

import (
	"context"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user User) error
	GetByID(ctx context.Context, id uuid.UUID) (User, error)
}

type ProfileRepository interface {
	Create(ctx context.Context, profile Profile) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (Profile, error)
}
