package userrepo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vlaner/go-backend-examples/logging/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, user domain.User, password string) (domain.User, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (id, username, password)
		VALUES (@id, @username, @password)
		RETURNING id, username
	`, pgx.NamedArgs{
		"id":       user.ID,
		"username": user.Username,
		"password": password,
	}).Scan(&user.ID, &user.Username)
	if err != nil {
		return domain.User{}, fmt.Errorf("insert user: %w", err)
	}

	return user, nil
}

func (r *Repository) CountByUsername(ctx context.Context, username string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM users
		WHERE username = @username
	`, pgx.NamedArgs{"username": username}).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count users by username: %w", err)
	}

	return count, nil
}
