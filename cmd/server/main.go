package main

import (
	"context"
	"errors"
	"log"
	"net/http"
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

	worker := scheduler.NewWorker(
		jobRepo,
		time.Duration(cfg.PollIntervalSec)*time.Second,
		cfg.WorkerCount,
	)
	go worker.Start(ctx)

	// reaper: runs every 30s, recovers jobs from workers that crashed
	// heartbeat fires every 10s — 30s means 3 missed heartbeats before a job is considered stale
	reaper := scheduler.NewReaper(jobRepo, 30*time.Second, 30*time.Second)
	go reaper.Start(ctx)

	srv := http.Server{
		Addr:    ":" + cfg.Port,
		Handler: httptransport.NewRouter(&jobHandler),
	}

	// run HTTP server in separate goroutine
	go func() {
		log.Printf("server listening on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	// block until Ctrl+C / SIGTERM
	<-ctx.Done()
	log.Println("shutting down...")

	// give in-flight HTTP requests up to 10s to finish
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown: %v", err)
	}
}
