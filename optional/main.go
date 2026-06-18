package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	pgcontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
)

const readHeaderTimeout = 5 * time.Second

func main() {
	ctx := context.Background()
	err := run(ctx)
	if err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	container, err := pgcontainer.Run(ctx,
		"postgres:18-alpine3.23",
		pgcontainer.WithDatabase("app"),
		pgcontainer.WithUsername("app"),
		pgcontainer.WithPassword("app"),
		pgcontainer.BasicWaitStrategies(),
	)
	if err != nil {
		return fmt.Errorf("run postgres container: %w", err)
	}
	defer func() {
		terminateErr := testcontainers.TerminateContainer(container)
		if terminateErr != nil {
			log.Printf("terminate postgres container: %v", terminateErr)
		}
	}()

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return fmt.Errorf("postgres connection string: %w", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()

	err = migrate(ctx, pool)
	if err != nil {
		return err
	}
	err = seed(ctx, pool)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	NewUserHandler(NewUserService(NewUserRepository(pool))).RegisterRoutes(mux)

	log.Println("try:")
	log.Println(`curl -X PATCH localhost:8080/users/1 -d '{}'`)
	log.Println(`curl -X PATCH localhost:8080/users/1 -d '{"bio":"new bio"}'`)
	log.Println(`curl -X PATCH localhost:8080/users/1 -d '{"bio":null}'`)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}
	err = server.ListenAndServe()
	if err != nil {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return nil
}

func migrate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE users (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			bio TEXT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("migrate optional example: %w", err)
	}

	return nil
}

func seed(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO users (name, bio) VALUES
			('Alice', 'original bio');
	`)
	if err != nil {
		return fmt.Errorf("seed optional example: %w", err)
	}

	return nil
}
