package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mmfshirokan/task-manager/config"
	"github.com/mmfshirokan/task-manager/internal/cron"
	"github.com/mmfshirokan/task-manager/internal/handlers"
	"github.com/mmfshirokan/task-manager/internal/repository"
	"github.com/mmfshirokan/task-manager/internal/service"
	"github.com/mmfshirokan/task-manager/schema"
)

const readHeaderTimeout = 5 * time.Second

// Run загружает конфигурацию, применяет миграции, связывает слои и обслуживает
// запросы до получения SIGINT/SIGTERM.
func Run() error {
	cfg, err := config.Load(".env")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	applied, err := schema.Migrate(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}

	logger.Info("migrations applied", "versions", applied)

	pool, err := connectDB(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	repo := repository.New(pool)
	svc := service.New(repo.Users, repo.Tasks, service.Config{JWTSecret: []byte(cfg.JWTSecret), TokenTTL: cfg.JWTTTL}, logger)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           handlers.New(svc.Auth, svc.Tasks, logger).Routes(),
		ReadHeaderTimeout: readHeaderTimeout,
	}
	job := cron.New(svc.Tasks, cfg.OverdueInterval, logger)

	return serve(ctx, srv, job, cfg.ShutdownTimeout, logger)
}

func connectDB(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("create db pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()

		return nil, fmt.Errorf("ping db: %w", err)
	}

	return pool, nil
}

// serve запускает HTTP-сервер и фоновую задачу и работает, пока не отменён ctx
// или не упал сервер; затем корректно останавливает оба.
func serve(ctx context.Context, srv *http.Server, job *cron.OverdueJob, timeout time.Duration, log *slog.Logger) error {
	jobCtx, cancelJob := context.WithCancel(ctx)
	defer cancelJob()

	var wg sync.WaitGroup

	wg.Go(func() { job.Run(jobCtx) })

	serverErr := make(chan error, 1)

	go func() {
		log.Info("http server started", "addr", srv.Addr)

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	var runErr error

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-serverErr:
		runErr = fmt.Errorf("http server: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		runErr = errors.Join(runErr, fmt.Errorf("shutdown http server: %w", err))
	}

	cancelJob()
	wg.Wait()

	log.Info("service stopped")

	return runErr
}
