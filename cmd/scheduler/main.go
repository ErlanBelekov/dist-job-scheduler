package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ErlanBelekov/dist-job-scheduler/config"
	"github.com/ErlanBelekov/dist-job-scheduler/internal/infrastructure/postgres"
	"github.com/ErlanBelekov/dist-job-scheduler/internal/metrics"
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

	metrics.Register()

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

	metricsSrv := metrics.NewServer(":" + cfg.MetricsPort)
	go func() {
		logger.Info("metrics server started", "port", cfg.MetricsPort)
		if err := metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("metrics server", "error", err)
		}
	}()

	<-ctx.Done()
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error("metrics server shutdown", "error", err)
	}

	logger.Info("scheduler shut down")
}

func newLogger(env string) *slog.Logger {
	if env == "local" {
		return slog.New(slog.NewTextHandler(os.Stdout, nil))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}
