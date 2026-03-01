package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ErlanBelekov/dist-job-scheduler/config"
	"github.com/ErlanBelekov/dist-job-scheduler/internal/infrastructure/postgres"
	"github.com/ErlanBelekov/dist-job-scheduler/internal/scheduler"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	logger := newLogger(cfg.Env)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		stop()
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	logger.Info("db connected")

	jobRepo := postgres.NewJobRepository(pool)
	attemptRepo := postgres.NewAttemptRepository(pool)

	worker := scheduler.NewWorker(
		jobRepo,
		attemptRepo,
		logger,
		time.Duration(cfg.PollIntervalSec)*time.Second,
		cfg.WorkerCount,
	)
	go worker.Start(ctx)

	// heartbeat fires every 10s — 30s timeout means 3 missed beats before a job is stale
	reaper := scheduler.NewReaper(jobRepo, logger, 30*time.Second, 30*time.Second)
	go reaper.Start(ctx)

	<-ctx.Done()
	stop()
	logger.Info("scheduler shut down")
}

func newLogger(env string) *slog.Logger {
	if env == "local" {
		return slog.New(slog.NewTextHandler(os.Stdout, nil))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}
