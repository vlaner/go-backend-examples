package main

import (
	"context"
	"fmt"

	"github.com/vlaner/go-backend-examples/pgx-transactions/postgres"
)

func migrate(ctx context.Context, db postgres.DBTX) error {
	_, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			username VARCHAR(255) NOT NULL UNIQUE,
			password TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS profiles (
			user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			description VARCHAR(255),
			contact VARCHAR(255),
			social_media_links VARCHAR(255)[]
		);
	`)
	if err != nil {
		return fmt.Errorf("migrate pgx transactions: %w", err)
	}

	return nil
}
