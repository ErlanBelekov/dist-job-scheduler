package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ErlanBelekov/dist-job-scheduler/config"
	"github.com/ErlanBelekov/dist-job-scheduler/internal/infrastructure/postgres"
	"github.com/ErlanBelekov/dist-job-scheduler/internal/scheduler"
	httptransport "github.com/ErlanBelekov/dist-job-scheduler/internal/transport/http"
	"github.com/ErlanBelekov/dist-job-scheduler/internal/transport/http/handler"
	"github.com/ErlanBelekov/dist-job-scheduler/internal/usecase"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	jobRepo := postgres.NewJobRepository(pool)
	jobUsecase := usecase.NewJobUsecase(jobRepo)
	jobHandler := handler.NewJobHandler(jobUsecase)

	// start worker in background
	worker := scheduler.NewWorker(
		jobRepo,
		time.Duration(cfg.PollIntervalSec)*time.Second,
		cfg.WorkerCount,
	)
	go worker.Start(ctx)

	r := httptransport.NewRouter(jobHandler)
	r.Run(":" + cfg.Port)
}
