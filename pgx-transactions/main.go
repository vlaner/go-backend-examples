package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/testcontainers/testcontainers-go"
	pgcontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	approachtransactionalservice "github.com/vlaner/go-backend-examples/pgx-transactions/approach-transactional-service"
	approachtxinctx "github.com/vlaner/go-backend-examples/pgx-transactions/approach-tx-in-ctx"
	approachunitofwork "github.com/vlaner/go-backend-examples/pgx-transactions/approach-unit-of-work"
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

	txInCtxExecutor := postgres.NewContextExecutor(pool)
	users := userrepo.New(txInCtxExecutor)
	profiles := profilerepo.New(txInCtxExecutor)
	txManager := postgres.NewTxManager(pool)

	txInCtxService := approachtxinctx.NewUserService(txManager, users, profiles)
	txInCtxResult, err := txInCtxService.CreateUserWithProfile(ctx, approachtxinctx.CreateUserWithProfileInput{
		Username:         "approach-tx-in-ctx",
		Password:         examplePassword,
		Description:      "context transaction profile",
		Contact:          "approach-tx-in-ctx@example.com",
		SocialMediaLinks: []string{"https://github.com/example/approach-tx-in-ctx"},
	})
	if err != nil {
		return fmt.Errorf("approach-tx-in-ctx create user with profile: %w", err)
	}
	logger.InfoContext(ctx, "approach-tx-in-ctx created user and profile", "user_id", txInCtxResult.User.ID)

	unitOfWorkService := approachunitofwork.NewUserService(postgres.NewPGXUnitOfWork(pool, approachunitofwork.NewRepositories))
	unitOfWorkResult, err := unitOfWorkService.CreateUserWithProfile(ctx, approachunitofwork.CreateUserWithProfileInput{
		Username:         "approach-unit-of-work",
		Password:         examplePassword,
		Description:      "unit of work profile",
		Contact:          "approach-unit-of-work@example.com",
		SocialMediaLinks: []string{"https://github.com/example/approach-unit-of-work"},
	})
	if err != nil {
		return fmt.Errorf("approach-unit-of-work create user with profile: %w", err)
	}
	logger.InfoContext(ctx, "approach-unit-of-work created user and profile", "user_id", unitOfWorkResult.User.ID)

	transactionalServiceApp := approachtransactionalservice.NewApplication(approachtransactionalservice.NewPGXTransactionalUserService(pool))
	transactionalServiceResult, err := transactionalServiceApp.CreateUserWithProfile(ctx, approachtransactionalservice.CreateUserWithProfileInput{
		Username:         "approach-transactional-service",
		Password:         examplePassword,
		Description:      "recreate service per request",
		Contact:          "approach-transactional-service@example.com",
		SocialMediaLinks: []string{"https://github.com/example/approach-transactional-service"},
	})
	if err != nil {
		return fmt.Errorf("approach-transactional-service create user with profile: %w", err)
	}
	logger.InfoContext(ctx, "approach-transactional-service created user and profile", "user_id", transactionalServiceResult.User.ID)

	// approach-tx-manager: same service, wrapped by a tx manager.
	// The base service uses a context executor, so the tx manager supplies the tx through ctx.
	contextExecutor := postgres.NewContextExecutor(pool)
	txManagerWithConfig := postgres.NewTxManager(pool)
	txManagerService := approachtransactionalservice.NewTxManagerUserService(
		approachtransactionalservice.NewUserService(userrepo.New(contextExecutor), profilerepo.New(contextExecutor)),
		approachtransactionalservice.Transactions{CreateUserWithProfile: txManagerWithConfig.WithConfig(postgres.SerializableTx())},
	)
	txManagerResult, err := txManagerService.CreateUserWithProfile(ctx, approachtransactionalservice.CreateUserWithProfileInput{
		Username:         "approach-tx-manager",
		Password:         examplePassword,
		Description:      "tx manager transaction profile",
		Contact:          "approach-tx-manager@example.com",
		SocialMediaLinks: []string{"https://github.com/example/approach-tx-manager"},
	})
	if err != nil {
		return fmt.Errorf("approach-tx-manager create user with profile: %w", err)
	}
	logger.InfoContext(ctx, "approach-tx-manager created user and profile", "user_id", txManagerResult.User.ID)

	// approach-unit-of-work-service: same service, wrapped by unit of work.
	// The unit of work starts the tx and rebuilds the service with tx-scoped repositories.
	unitOfWorkWrapperService := approachtransactionalservice.NewUnitOfWorkUserService(
		postgres.NewPGXUnitOfWork(pool, func(db postgres.DBTX) approachtransactionalservice.UserService {
			return approachtransactionalservice.NewUserService(userrepo.New(db), profilerepo.New(db))
		}),
	)
	unitOfWorkWrapperResult, err := unitOfWorkWrapperService.CreateUserWithProfile(ctx, approachtransactionalservice.CreateUserWithProfileInput{
		Username:         "approach-unit-of-work-service",
		Password:         examplePassword,
		Description:      "unit of work transaction profile",
		Contact:          "approach-unit-of-work-service@example.com",
		SocialMediaLinks: []string{"https://github.com/example/approach-unit-of-work-service"},
	})
	if err != nil {
		return fmt.Errorf("approach-unit-of-work-service create user with profile: %w", err)
	}
	logger.InfoContext(ctx, "approach-unit-of-work-service created user and profile", "user_id", unitOfWorkWrapperResult.User.ID)

	return nil
}
