package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bekonat/taskflow/internal/auth"
	"github.com/bekonat/taskflow/internal/config"
	"github.com/bekonat/taskflow/internal/repository/memory"
	"github.com/bekonat/taskflow/internal/repository/postgres"
	"github.com/bekonat/taskflow/internal/service"
	"github.com/bekonat/taskflow/internal/transport/httpapi"
	"github.com/bekonat/taskflow/migrations"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("fatal", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var (
		userRepo service.UserRepository
		taskRepo service.TaskRepository
		health   httpapi.HealthChecker
	)

	if cfg.DatabaseURL != "" {
		pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()

		if err := pool.Ping(ctx); err != nil {
			return err
		}
		if err := postgres.Migrate(ctx, pool, migrations.FS); err != nil {
			return err
		}
		userRepo = postgres.NewUserRepository(pool)
		taskRepo = postgres.NewTaskRepository(pool)
		health = pool
		log.Info("using postgres storage")
	} else {
		userRepo = memory.NewUserRepository()
		taskRepo = memory.NewTaskRepository()
		log.Warn("DATABASE_URL is not set, using in-memory storage")
	}

	jwt := auth.NewJWTManager(cfg.JWTSecret, cfg.TokenTTL)
	users := service.NewUserService(userRepo, auth.NewBcryptHasher(), jwt)
	tasks := service.NewTaskService(taskRepo)
	handler := httpapi.NewHandler(users, tasks, jwt, health, log)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("http server started", slog.String("addr", cfg.HTTPAddr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
