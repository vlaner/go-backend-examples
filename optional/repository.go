package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type userRow struct {
	ID   int64          `db:"id"`
	Name string         `db:"name"`
	Bio  sql.NullString `db:"bio"`
}

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Find(ctx context.Context, id int64) (User, error) {
	return scanUser(r.pool.QueryRow(ctx, `SELECT id, name, bio FROM users WHERE id = $1`, id))
}

func (r *UserRepository) UpdateBio(ctx context.Context, id int64, bio *string) (User, error) {
	return scanUser(r.pool.QueryRow(ctx, `
		UPDATE users
		SET bio = $2
		WHERE id = $1
		RETURNING id, name, bio
	`, id, bio))
}

func scanUser(row pgx.Row) (User, error) {
	var dbUser userRow
	err := row.Scan(&dbUser.ID, &dbUser.Name, &dbUser.Bio)
	if err != nil {
		return User{}, fmt.Errorf("scan user: %w", err)
	}

	return mapUserRow(dbUser), nil
}

func mapUserRow(dbUser userRow) User {
	user := User{
		ID:   dbUser.ID,
		Name: dbUser.Name,
	}
	if dbUser.Bio.Valid {
		user.Bio = &dbUser.Bio.String
	}

	return user
}
