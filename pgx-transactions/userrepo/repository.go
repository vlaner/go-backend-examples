package userrepo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/vlaner/go-backend-examples/pgx-transactions/domain"
	"github.com/vlaner/go-backend-examples/pgx-transactions/postgres"
)

type dbUser struct {
	ID       uuid.UUID `db:"id"`
	Username string    `db:"username"`
	Password string    `db:"password"`
}

func (u dbUser) ToDomain() domain.User {
	return domain.User{
		ID:       u.ID,
		Username: u.Username,
		Password: u.Password,
	}
}

type repository struct {
	db postgres.DBTX
}

func New(db postgres.DBTX) domain.UserRepository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, user domain.User) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO users (id, username, password)
		VALUES ($1, $2, $3)
	`, user.ID, user.Username, user.Password)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, username, password
		FROM users
		WHERE id = $1
	`, id)
	if err != nil {
		return domain.User{}, fmt.Errorf("query user by id: %w", err)
	}

	user, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[dbUser])
	if err != nil {
		return domain.User{}, fmt.Errorf("collect user by id: %w", err)
	}

	return user.ToDomain(), nil
}
