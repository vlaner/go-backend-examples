package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/testcontainers/testcontainers-go"
	pgcontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/vlaner/go-backend-examples/pgx-transactions/approach1"
	"github.com/vlaner/go-backend-examples/pgx-transactions/approach2"
	"github.com/vlaner/go-backend-examples/pgx-transactions/approach3"
	"github.com/vlaner/go-backend-examples/pgx-transactions/manager"
	"github.com/vlaner/go-backend-examples/pgx-transactions/postgres"
	"github.com/vlaner/go-backend-examples/pgx-transactions/profilerepo"
	"github.com/vlaner/go-backend-examples/pgx-transactions/userrepo"
)

const examplePassword = "secret"

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

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pool, err := postgres.Connect(ctx, dsn, postgres.WithTracer(postgres.NewLoggingQueryTracer(logger)))
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()

	err = migrate(ctx, pool)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	approach1Executor := postgres.NewContextExecutor(pool)
	users := userrepo.New(approach1Executor)
	profiles := profilerepo.New(approach1Executor)
	txManager := manager.NewPGXManager(pool)

	approach1Service := approach1.NewUserService(txManager, users, profiles)
	approach1Result, err := approach1Service.CreateUserWithProfile(ctx, approach1.CreateUserWithProfileInput{
		Username:         "approach-1",
		Password:         examplePassword,
		Description:      "context transaction profile",
		Contact:          "approach-1@example.com",
		SocialMediaLinks: []string{"https://github.com/example/approach-1"},
	})
	if err != nil {
		return fmt.Errorf("approach 1 create user with profile: %w", err)
	}
	logger.InfoContext(ctx, "approach 1 created user and profile", "user_id", approach1Result.User.ID)

	approach2Service := approach2.NewUserService(postgres.NewPGXUnitOfWork(pool, approach2.NewRepositories))
	approach2Result, err := approach2Service.CreateUserWithProfile(ctx, approach2.CreateUserWithProfileInput{
		Username:         "approach-2",
		Password:         examplePassword,
		Description:      "unit of work profile",
		Contact:          "approach-2@example.com",
		SocialMediaLinks: []string{"https://github.com/example/approach-2"},
	})
	if err != nil {
		return fmt.Errorf("approach 2 create user with profile: %w", err)
	}
	logger.InfoContext(ctx, "approach 2 created user and profile", "user_id", approach2Result.User.ID)

	approach3App := approach3.NewApplication(approach3.NewPGXTransactionalUserService(pool))
	approach3Result, err := approach3App.CreateUserWithProfile(ctx, approach3.CreateUserWithProfileInput{
		Username:         "approach-3",
		Password:         examplePassword,
		Description:      "recreate service per request",
		Contact:          "approach-3@example.com",
		SocialMediaLinks: []string{"https://github.com/example/approach-3"},
	})
	if err != nil {
		return fmt.Errorf("approach 3 create user with profile: %w", err)
	}
	logger.InfoContext(ctx, "approach 3 created user and profile", "user_id", approach3Result.User.ID)

	// Approach 3B: same service, wrapped by a tx manager.
	// The base service uses a context executor, so the tx manager supplies the tx through ctx.
	contextExecutor := postgres.NewContextExecutor(pool)
	txManagerWithConfig := postgres.NewTxManager(pool)
	approach3TxManagerService := approach3.NewTxManagerUserService(
		approach3.NewUserService(userrepo.New(contextExecutor), profilerepo.New(contextExecutor)),
		approach3.Transactions{CreateUserWithProfile: txManagerWithConfig.WithConfig(postgres.SerializableTx())},
	)
	approach3TxManagerResult, err := approach3TxManagerService.CreateUserWithProfile(ctx, approach3.CreateUserWithProfileInput{
		Username:         "approach-3-tx-manager",
		Password:         examplePassword,
		Description:      "tx manager transaction profile",
		Contact:          "approach-3-tx-manager@example.com",
		SocialMediaLinks: []string{"https://github.com/example/approach-3-tx-manager"},
	})
	if err != nil {
		return fmt.Errorf("approach 3 tx manager create user with profile: %w", err)
	}
	logger.InfoContext(ctx, "approach 3 tx manager created user and profile", "user_id", approach3TxManagerResult.User.ID)

	// Approach 3C: same service, wrapped by unit of work.
	// The unit of work starts the tx and rebuilds the service with tx-scoped repositories.
	approach3UnitOfWorkService := approach3.NewUnitOfWorkUserService(
		postgres.NewPGXUnitOfWork(pool, func(db postgres.DBTX) approach3.UserService {
			return approach3.NewUserService(userrepo.New(db), profilerepo.New(db))
		}),
	)
	approach3UnitOfWorkResult, err := approach3UnitOfWorkService.CreateUserWithProfile(ctx, approach3.CreateUserWithProfileInput{
		Username:         "approach-3-unit-of-work",
		Password:         examplePassword,
		Description:      "unit of work transaction profile",
		Contact:          "approach-3-unit-of-work@example.com",
		SocialMediaLinks: []string{"https://github.com/example/approach-3-unit-of-work"},
	})
	if err != nil {
		return fmt.Errorf("approach 3 unit of work create user with profile: %w", err)
	}
	logger.InfoContext(ctx, "approach 3 unit of work created user and profile", "user_id", approach3UnitOfWorkResult.User.ID)

	return nil
}
